package mail

import (
	"bytes"
	"sync/atomic"
)

type Mail struct {
	From    string
	To      To
	Message Message
}

type Recorder interface {
	Sender
	Seen(Mail) (ok bool, err error)
}

type ConfigurableRecorder interface {
	ConfigurableSender
	Recorder
	TxConfig() *TxConfig
}

type MemRecorder struct {
	Mails []Mail
	cfg   atomic.Value
}

func (r *MemRecorder) Seen(m Mail) (ok bool, err error) {
	if len(r.Mails) == 0 {
		return false, nil
	}

	for _, r := range r.Mails {
		if r.From != m.From {
			return false, nil
		}

		if len(r.To) != len(m.To) {
			return false, nil
		}

		for i, to := range r.To {
			if to != m.To[i] {
				return false, nil
			}
		}

		if r.Message.Body != m.Message.Body ||
			r.Message.ContentType != m.Message.ContentType ||
			r.Message.Topic != m.Message.Topic {
			return false, nil
		}

		if len(r.Message.Attachments) != len(m.Message.Attachments) {
			return false, nil
		}

		for i, a := range r.Message.Attachments {
			a2 := m.Message.Attachments[i]
			if !bytes.Equal(a.Content, a2.Content) ||
				a.Kind != a2.Kind ||
				a.Name != a2.Name {
				return false, nil
			}
		}
	}

	return true, nil
}

func (r *MemRecorder) Send(from string, to To, message Message, options ...SendOption) (err error) {
	r.Mails = append(r.Mails, Mail{
		From:    from,
		To:      to,
		Message: message,
	})

	return nil
}

func (r *MemRecorder) UpdateTxConfig(cfg TxConfig) {
	r.cfg.Store(cfg)
}

func (r *MemRecorder) TxConfig() TxConfig {
	cfg, _ := r.cfg.Load().(TxConfig)
	return cfg
}
