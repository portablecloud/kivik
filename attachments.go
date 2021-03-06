package kivik

import (
	"bytes"
	"io"
)

// Checksum is a 128-bit MD5 checksum of a file's content.
type Checksum [16]byte

// Attachment represents a CouchDB file attachment.
type Attachment struct {
	io.ReadCloser
	Filename    string
	ContentType string
	MD5         Checksum
}

// bufCloser wraps a *bytes.Buffer to create an io.ReadCloser
type bufCloser struct {
	*bytes.Buffer
}

var _ io.ReadCloser = &bufCloser{}

func (b *bufCloser) Close() error { return nil }

// Bytes returns the attachment's body as a byte slice.
func (a *Attachment) Bytes() ([]byte, error) {
	if buf, ok := a.ReadCloser.(*bufCloser); ok {
		// Simple optimization
		return buf.Bytes(), nil
	}
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, a); err != nil {
		return nil, err
	}
	a.ReadCloser = &bufCloser{buf}
	return buf.Bytes(), nil
}

// NewAttachment returns a new CouchDB attachment.
func NewAttachment(filename, contentType string, body io.ReadCloser) *Attachment {
	return &Attachment{
		ReadCloser:  body,
		Filename:    filename,
		ContentType: contentType,
	}
}
