package main

import (
	"bytes"
	_context "context"
	"fmt"
	"strings"

	"net/http"
	"text/template"

	"github.com/ysugimoto/falco/context"
	"github.com/ysugimoto/falco/lexer"
	"github.com/ysugimoto/falco/linter"
	"github.com/ysugimoto/falco/parser"
	"github.com/ysugimoto/falco/remote"
)

var tableTemplate = `
table {{ .Name }} {
	{{- range .Items }}
	"{{ .Key }}": "{{ .Value }}",
	{{- end }}
}
`

var aclTemplate = `
acl {{ .Name }} {
	{{- range .Entries }}
	{{ if .Negated }}!{{ end }}"{{ .Ip }}"{{ if .Subnet }}/{{ .Subnet }}{{ end }};{{ if .Comment }}  # {{ .Comment }}{{ end }}
	{{- end }}
}
`

var backendTemplate = `
backend F_{{ .Name }} {}
`

var directorTemplate = `
director {{ .Name }} {{ .Type | printtype }} {
	{{- range .Backends }}
	{ .backend = {{ . }}; .weight = 1; }
	{{- end }}
}
`

type Snippet struct {
	client   *remote.FastlyClient
	snippets []string
}

func NewSnippet(serviceId, apiKey string) *Snippet {
	return &Snippet{
		client: remote.NewFastlyClient(http.DefaultClient, serviceId, apiKey),
	}
}

func (s *Snippet) Compile(ctx *context.Context) error {
	vcl, err := parser.New(lexer.NewFromString(strings.Join(s.snippets, "\n"))).ParseVCL()
	if err != nil {
		return err
	}
	l := linter.New()
	l.Lint(vcl, ctx, false)
	return nil
}

func (s *Snippet) Fetch(c _context.Context) error {
	// Fetch latest version
	version, err := s.client.LatestVersion(c)
	if err != nil {
		return fmt.Errorf("Failed to get latest version %w", err)
	}
	write(white, "Fetching Edge Dictionaries...")
	dicts, err := s.fetchEdgeDictionary(c, version)
	if err != nil {
		return err
	}
	writeln(white, "Done")
	s.snippets = append(s.snippets, dicts...)

	write(white, "Fatching Access Control Lists...")
	acls, err := s.fetchAccessControl(c, version)
	if err != nil {
		return err
	}
	writeln(white, "Done")
	s.snippets = append(s.snippets, acls...)

	write(white, "Fatching Backends...")
	backends, err := s.fetchBackend(c, version)
	if err != nil {
		return err
	}
	writeln(white, "Done")
	s.snippets = append(s.snippets, backends...)

	return nil
}

// Fetch remote Edge dictionary items
func (s *Snippet) fetchEdgeDictionary(c _context.Context, version int64) ([]string, error) {
	dicts, err := s.client.ListEdgeDictionaries(c, version)
	if err != nil {
		return nil, fmt.Errorf("Failed to get edge dictionaries for version %w", err)
	}

	tmpl, err := template.New("table").Parse(tableTemplate)
	if err != nil {
		return nil, fmt.Errorf("Failed to compilte table template: %w", err)
	}

	var snippets []string
	for _, dict := range dicts {
		buf := new(bytes.Buffer)
		if err := tmpl.Execute(buf, dict); err != nil {
			return nil, fmt.Errorf("Failed to render table template: %w", err)
		}
		snippets = append(snippets, buf.String())
	}
	return snippets, nil
}

// Fetch remote Access Control entries
func (s *Snippet) fetchAccessControl(c _context.Context, version int64) ([]string, error) {
	acls, err := s.client.ListAccessControlLists(c, version)
	if err != nil {
		return nil, fmt.Errorf("Failed to get access control lists for version %w", err)
	}

	tmpl, err := template.New("acl").Parse(aclTemplate)
	if err != nil {
		return nil, fmt.Errorf("Failed to compilte acl template: %w", err)
	}

	var snippets []string
	for _, a := range acls {
		buf := new(bytes.Buffer)
		if err := tmpl.Execute(buf, a); err != nil {
			return nil, fmt.Errorf("Failed to render table template: %w", err)
		}
		snippets = append(snippets, buf.String())
	}
	return snippets, nil
}

func (s *Snippet) fetchBackend(c _context.Context, version int64) ([]string, error) {
	backends, err := s.client.ListBackends(c, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get backends for version %w", err)
	}

	backTmpl, err := template.New("backend").Parse(backendTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to compile backend template: %w", err)
	}

	var snippets []string
	for _, b := range backends {
		buf := new(bytes.Buffer)
		if err := backTmpl.Execute(buf, b); err != nil {
			return nil, fmt.Errorf("failed to render backend template: %w", err)
		}
		snippets = append(snippets, buf.String())
	}

	directors, err := s.renderBackendShields(backends)
	if err != nil {
		return nil, err
	}
	snippets = append(snippets, directors...)

	return snippets, nil
}

func (s *Snippet) renderBackendShields(backends []*remote.Backend) ([]string, error) {
	printType := func(dtype remote.DirectorType) string {
		switch dtype {
		case remote.Random:
			return "random"
		case remote.Hash:
			return "hash"
		case remote.Client:
			return "client"
		}
		return ""
	}
	dirTmpl, err := template.New("director").Funcs(template.FuncMap{"printtype": printType}).Parse(directorTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to compile director template: %w", err)
	}

	shieldDirectors := make(map[string]struct{})
	for _, b := range backends {
		if b.Shield != nil {
			shieldDirectors[*b.Shield] = struct{}{}
		}
	}

	var snippets []string
	// We need to pick an arbitrary backend to avoid an undeclared linter error
	shieldBackend := "F_" + backends[0].Name
	for sd := range shieldDirectors {
		d := remote.Director{
			Name:     "ssl_shield_" + strings.ReplaceAll(sd, "-", "_"),
			Type:     remote.Random,
			Backends: []string{shieldBackend},
		}
		buf := new(bytes.Buffer)
		if err := dirTmpl.Execute(buf, d); err != nil {
			return nil, fmt.Errorf("failed to render director template: %w", err)
		}
		snippets = append(snippets, buf.String())
	}

	return snippets, nil
}
