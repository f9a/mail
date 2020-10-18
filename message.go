package mail

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

// Attachment is attachment for message send via smtp server
type Attachment struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Content []byte `json:"content"`
}

// Message is message send via smtp server
type Message struct {
	Topic       string       `json:"topic"`
	Body        string       `json:"body"`
	Attachments []Attachment `json:"attachments"`
	ContentType string       `json:"contentType"`
}

// RequestAttachment can be used in the request struct when attachments are allowed
type RequestAttachment struct {
	Name string `json:"name"`
	// Content base64 encoded content
	Content []byte `json:"content"`
}

// RequestAttachments list of RequestAttachments
type RequestAttachments []RequestAttachment

// Template template for message
type Template struct {
	topic                  *template.Template
	body                   *template.Template
	allowedAttachmentTypes map[string]struct{}
	funcs                  template.FuncMap
	contentType            string
	attachments            RequestAttachments
}

func processAttachments(
	allowed map[string]struct{},
	attachments RequestAttachments,
) (aa []Attachment, err error) {
	for _, attachment := range attachments {
		mimeType := http.DetectContentType(attachment.Content)
		if _, ok := allowed[mimeType]; !ok {
			return aa, fmt.Errorf("MIME Type %v is not allowed", mimeType)
		}

		aa = append(aa, Attachment{
			Name:    attachment.Name,
			Kind:    mimeType,
			Content: attachment.Content,
		})
	}

	return
}

func executeTemplate(tpl *template.Template, data interface{}) (s string, err error) {
	var buf strings.Builder

	err = tpl.Execute(&buf, data)
	if err != nil {
		return
	}

	return buf.String(), err
}

// WithAttachments add attacments to a message
func WithAttachments(attachments RequestAttachments) Option {
	return func(tpl *Template) {
		tpl.attachments = attachments
	}
}

// Execute builds message with given data and options
func (tpl Template) Execute(data interface{}, opts ...Option) (msg Message, err error) {
	for _, opt := range opts {
		opt(&tpl)
	}

	topic, err := executeTemplate(tpl.topic, data)
	if err != nil {
		return
	}
	msg.Topic = topic

	body, err := executeTemplate(tpl.body, data)
	if err != nil {
		return
	}
	msg.Body = body

	var messageAttachments []Attachment
	messageAttachments, err = processAttachments(
		tpl.allowedAttachmentTypes,
		tpl.attachments,
	)
	if err != nil {
		err = fmt.Errorf("wrong attachment: %v", err)
		return
	}

	msg.Attachments = messageAttachments
	msg.ContentType = tpl.contentType

	return
}

// Option option to configure template
type Option func(*Template)

// AllowAttachments allows attachements for letter
func AllowAttachments(types ...string) Option {
	return func(opts *Template) {
		opts.allowedAttachmentTypes = makeAllowedAttachmentTypesIdx(types)
	}
}

// ContentType is content-type of message
func ContentType(kind string) Option {
	return func(opts *Template) {
		opts.contentType = kind
	}
}

// TemplateFuncs merge template funcs with default template funcs for letter
func TemplateFuncs(funcs template.FuncMap) Option {
	return func(opts *Template) {
		for k, fun := range funcs {
			opts.funcs[k] = fun
		}
	}
}

var days = map[time.Weekday]string{
	time.Monday:    "Montag",
	time.Tuesday:   "Dienstag",
	time.Wednesday: "Mittwoch",
	time.Thursday:  "Donnerstag",
	time.Friday:    "Freitag",
	time.Saturday:  "Samstag",
	time.Sunday:    "Sonntag",
}

var months = map[time.Month]string{
	time.January:   "Januar",
	time.February:  "Februar",
	time.March:     "März",
	time.April:     "April",
	time.May:       "Mai",
	time.June:      "Juni",
	time.July:      "Juli",
	time.August:    "August",
	time.September: "September",
	time.October:   "Oktober",
	time.November:  "November",
	time.December:  "Dezember",
}

func longDateGerman(t time.Time) string {
	day := days[t.Weekday()]
	month := months[t.Month()]

	return fmt.Sprintf("%s, %02d. %s %d", day[:2], t.Day(), month, t.Year())
}

func longTimeGerman(t time.Time) string {
	return fmt.Sprintf("%s %02d:%02d:%02d", longDateGerman(t), t.Hour(), t.Minute(), t.Second())
}

func timef(t time.Time, format string) string {
	switch format {
	case "date-short-de":
		return t.Format("02.01.2006")
	case "date-long-de":
		return longDateGerman(t)
	case "time-short-de":
		return t.Format("02.01.2006 15:04:05")
	case "time-long-de":
		return longTimeGerman(t)
	default:
		return t.Format(format)
	}

}

func makeAllowedAttachmentTypesIdx(types []string) map[string]struct{} {
	idx := map[string]struct{}{}
	for _, t := range types {
		idx[t] = struct{}{}
	}

	return idx
}

// NewTemplate creates new template
func NewTemplate(topic, body string, options ...Option) (tpl Template, err error) {
	tpl.contentType = "text/plain"
	tpl.funcs = template.FuncMap{
		"timef": timef,
	}

	for _, option := range options {
		option(&tpl)
	}

	if tpl.allowedAttachmentTypes == nil {
		tpl.allowedAttachmentTypes = map[string]struct{}{}
	}

	tpl.topic, err = template.New("subject").Funcs(tpl.funcs).Parse(topic)
	if err != nil {
		return
	}

	tpl.body, err = template.New("body").Funcs(tpl.funcs).Parse(body)
	if err != nil {
		return
	}

	return
}
