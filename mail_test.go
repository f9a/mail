package mail_test

import (
	"testing"

	"github.com/f9a/mail"
)

type testData struct {
	Name  string
	Quote string
}

func TestMail(t *testing.T) {
	m, err := mail.Dial(mail.TxConfig{
		User:     "test@example.de",
		Password: "xxx",
		Host:     "smtp.example.de",
		Port:     38145,
		TmpDir:   "/tmp",
	})
	if err != nil {
		t.Fatal(err)
	}

	tpl, err := mail.NewTemplate("{{.Name}} says hello!", "{{.Quote}}")
	if err != nil {
		t.Fatal(err)
	}

	data := testData{
		Name:  "The Frenchman",
		Quote: "Quelle fantastique bugette",
	}

	msg, err := tpl.Execute(data)
	if err != nil {
		t.Fatal(err)
	}

	err = m.Send("test@example.de", mail.To{"ava@example.de"}, msg)
	if err != nil {
		t.Fatal(err)
	}
}
