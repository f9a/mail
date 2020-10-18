package mail

import (
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"
	"sync/atomic"

	oz "github.com/go-ozzo/ozzo-validation/v4"
	"gopkg.in/mail.v2"
)

type Sender interface {
	Send(from string, to To, message Message, options ...SendOption) (err error)
}

type ConfigurableSender interface {
	Sender
	// UpdateTxConfig must be safe for concurrenct use
	UpdateTxConfig(cfg TxConfig)
}

var _ Sender = &Tx{}

var _ ConfigurableSender = &Tx{}

type TxConfig struct {
	User     string `json:"user" ini:"user" yaml:"user"`
	Password string `json:"password" ini:"password" yaml:"password"`
	Host     string `json:"host" ini:"host" yaml:"host"`
	Port     int    `json:"port" ini:"port" yaml:"port"`
	TmpDir   string `json:"tmpDir" ini:"tmp-dir" envconfig:"TMP_DIR" yaml:"tmpDir"`
}

func (cfg TxConfig) Validate() error {
	return oz.ValidateStruct(&cfg,
		oz.Field(&cfg.User, oz.Required),
		oz.Field(&cfg.Password, oz.Required),
		oz.Field(&cfg.Host, oz.Required),
		oz.Field(&cfg.Port, oz.Required, oz.Min(0), oz.Max(49151)),
	)
}

type Tx struct {
	dialer atomic.Value
	cfg    atomic.Value
}

// To represents to addresses
type To []string

func writeFile(tempDirName string, a Attachment) (filename string, err error) {
	ee, err := mime.ExtensionsByType(a.Kind)
	if err != nil {
		err = fmt.Errorf("Couldn't find extension for mime-type: %v", err)
		return
	}

	var ext string
	if ee == nil {
		ext = "unknown"
	} else {
		ext = ee[0]
	}

	filename = filepath.Join(tempDirName, fmt.Sprintf("%s%s", a.Name, ext))
	err = ioutil.WriteFile(filename, a.Content, 0700)
	if err != nil {
		err = fmt.Errorf("Couldn't write attachment to tmp-dir: %v", err)
		return
	}

	return
}

type sendOptions struct {
	asCc bool
}

type SendOption interface {
	apply(*sendOptions)
}

type sendOptionFunc func(*sendOptions)

func (fun sendOptionFunc) apply(o *sendOptions) {
	fun(o)
}

func AsCc() SendOption {
	return sendOptionFunc(func(o *sendOptions) {
		o.asCc = true
	})
}

// Send sends message
func (tx *Tx) Send(from string, to To, message Message, options ...SendOption) (err error) {
	if from == "" {
		return errors.New("from cannot be empty")
	}

	if len(to) == 0 {
		return errors.New("at least one 'to' email-address must be given")
	}

	opts := sendOptions{}
	for _, o := range options {
		o.apply(&opts)
	}

	cfg, ok := tx.cfg.Load().(TxConfig)
	if !ok {
		err = errors.New("transmitter is not configured, yet")
		return
	}

	m := mail.NewMessage()

	m.SetHeader("From", from)
	m.SetHeader("To", to[0])
	if len(to) > 1 {
		if opts.asCc {
			m.SetHeader("Cc", to[1:]...)
		} else {
			m.SetHeader("Bcc", to[1:]...)
		}
	}
	m.SetHeader("Subject", message.Topic)
	m.SetBody(message.ContentType, message.Body)

	tempDirName, err := ioutil.TempDir(cfg.TmpDir, "f9a-mail")
	if err != nil {
		return fmt.Errorf("couldn't create tmp-dir for attachments: %v", err)
	}
	defer func() {
		err = os.RemoveAll(tempDirName)
	}()

	for _, a := range message.Attachments {
		filename, err := writeFile(tempDirName, a)
		if err != nil {
			return err
		}

		m.Attach(filename)
	}

	dialer, ok := tx.dialer.Load().(*mail.Dialer)
	if !ok {
		err = errors.New("transmitter is not configured, yet")
		return
	}

	err = dialer.DialAndSend(m)

	return
}

// UpdateTxConfig tx config. Is safe for concurrenct use.
func (tx *Tx) UpdateTxConfig(cfg TxConfig) {
	tx.cfg.Store(cfg)
	tx.dialer.Store(mail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password))
}

// Dial creates a new smtp transmitter and creates a dialer with passed config.
func Dial(cfg TxConfig) (tx *Tx, err error) {
	tx = &Tx{}
	if err = cfg.Validate(); err != nil {
		return
	}

	tx.cfg.Store(cfg)
	tx.dialer.Store(mail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password))

	return
}

// New creates a new smtp transmitter
func New() (tx *Tx) {
	tx = &Tx{}
	return
}
