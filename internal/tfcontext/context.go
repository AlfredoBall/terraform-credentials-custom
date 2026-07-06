package tfcontext

import (
	"errors"
	"strings"
)

type Context struct {
	Raw  string
	Type string
	Org  string
}

func Parse(raw string) (Context, error) {

	raw = strings.TrimSpace(raw)

	if raw == "" || raw == "default" {
		return Context{
			Raw:  "default",
			Type: "default",
			Org:  "",
		}, nil
	}

	parts := strings.Split(raw, ":")

	if len(parts) != 2 {
		return Context{}, errors.New("invalid TF_CONTEXT format (expected type:org or default)")
	}

	return Context{
		Raw:  raw,
		Type: parts[0],
		Org:  parts[1],
	}, nil
}
