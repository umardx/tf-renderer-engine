package renderer

import (
	"bytes"
	"embed"
	"text/template"
)

//go:embed s3_bucket.tf.tpl
var templates embed.FS

func RenderS3Bucket(data interface{}) (string, error) {
	tmplContent, err := templates.ReadFile("s3_bucket.tf.tpl")
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("s3").Parse(string(tmplContent))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
