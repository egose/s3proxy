package namespace

import (
	"errors"
	"net/url"
	"strings"
	"text/template"
	"text/template/parse"
)

var ErrMapping = errors.New("listing requires a reversible prefix mapping")
var ErrQuery = errors.New("invalid listing query")

type Mapping struct {
	VisibleBucket string
	Prefix        string
}

func ValidateTemplate(t *template.Template) error {
	if t == nil || t.Tree == nil || t.Tree.Root == nil || len(t.Templates()) != 1 {
		return ErrMapping
	}
	nodes := t.Tree.Root.Nodes
	keySeen := false
	for i, node := range nodes {
		switch n := node.(type) {
		case *parse.TextNode:
		case *parse.ActionNode:
			field := templateField(n)
			switch {
			case len(field) == 1 && field[0] == "Key":
				if keySeen || i != len(nodes)-1 {
					return ErrMapping
				}
				keySeen = true
			case len(field) == 1 && field[0] == "Bucket":
			case len(field) == 2 && field[0] == "Captures":
			default:
				return ErrMapping
			}
		default:
			return ErrMapping
		}
	}
	if !keySeen {
		return ErrMapping
	}
	return nil
}

func templateField(n *parse.ActionNode) []string {
	p := n.Pipe
	if p == nil || len(p.Decl) != 0 || len(p.Cmds) != 1 || len(p.Cmds[0].Args) != 1 {
		return nil
	}
	if field, ok := p.Cmds[0].Args[0].(*parse.FieldNode); ok {
		return field.Ident
	}
	return nil
}

func RawTemplatePrefix(t *template.Template, bucket string, captures map[string]string) (string, error) {
	if err := ValidateTemplate(t); err != nil {
		return "", err
	}
	var b strings.Builder
	for _, node := range t.Tree.Root.Nodes {
		switch n := node.(type) {
		case *parse.TextNode:
			b.Write(n.Text)
		case *parse.ActionNode:
			field := templateField(n)
			switch field[0] {
			case "Bucket":
				b.WriteString(bucket)
			case "Captures":
				value, ok := captures[field[1]]
				if !ok {
					return "", ErrMapping
				}
				b.WriteString(value)
			}
		}
	}
	return b.String(), nil
}

func New(visibleBucket, rawPrefix string) (*Mapping, error) {
	prefix, err := url.PathUnescape(rawPrefix)
	if err != nil {
		return nil, ErrMapping
	}
	return &Mapping{VisibleBucket: visibleBucket, Prefix: prefix}, nil
}

func (m Mapping) Query(q url.Values) (url.Values, error) {
	out := make(url.Values, len(q)+1)
	for k, v := range q {
		out[k] = append([]string(nil), v...)
	}
	for _, k := range []string{"prefix", "start-after", "encoding-type", "delimiter", "continuation-token"} {
		if len(q[k]) > 1 {
			return nil, ErrQuery
		}
	}
	if encoding := q.Get("encoding-type"); encoding != "" && encoding != "url" {
		return nil, ErrQuery
	}
	out.Set("prefix", m.Prefix+q.Get("prefix"))
	if q.Has("start-after") {
		out.Set("start-after", m.Prefix+q.Get("start-after"))
	}
	return out, nil
}
