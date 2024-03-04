package services

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"path"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/tiny-blob/tinyblob/config"
	"github.com/tiny-blob/tinyblob/pkg/funcmap"
)

type (
	// TemplateRenderer provides a flexible and easy to use method of rendering
	// simple templates or complex sets of templates while also providing caching
	// and/or hot-reloading depending on your current environment
	TemplateRenderer struct {
		// cfg stores the application configuration
		config *config.Config
		// funcMap stores the template function map
		funcMap template.FuncMap
		// templateCache stores a cache of the parsed page templates
		templateCache sync.Map
		// templatesPath stores the complete path to the templates directory
		templatesPath string
	}
	// TemplateParsed is a wrapper around parsed templates which are stored in the
	// TemplateRenderer cache.
	TemplateParsed struct {
		// build stores the build data used to parse the template
		build *templateBuild
		// Template stores the parsed template
		Template *template.Template
	}
	// templateBuild stores the build data used to parse a template.
	templateBuild struct {
		base        string
		directories []string
		files       []string
		group       string
		key         string
	}
	// templateBuilder handles chaining a template parse operation.
	templateBuilder struct {
		build    *templateBuild
		renderer *TemplateRenderer
	}
)

// NewTemplateRenderer creates a new TemplateRenderer
func NewTemplateRenderer(cfg *config.Config) *TemplateRenderer {
	t := &TemplateRenderer{
		config:        cfg,
		funcMap:       funcmap.GetFuncMap(),
		templateCache: sync.Map{},
	}

	// Gets the complete templates directory path.
	// We need to do this incase we call this function from outside our main
	// function i.e. tests.
	_, file, _, _ := runtime.Caller(0)
	d := path.Join(path.Dir(file))
	t.templatesPath = filepath.Join(filepath.Dir(d), config.TemplateDir)

	return t
}

// GetTemplatePath gets the complete path to the templates directory.
func (t *TemplateRenderer) GetTemplatesPath() string {
	return t.templatesPath
}

// Base sets the name of the base template to be used during parsing and
// execution. This should be only the file name without a directory or
// extension.
func (t *templateBuilder) Base(base string) *templateBuilder {
	t.build.base = base
	return t
}

// Directories sets the list of directories that all template files within will be parsed.
// The paths should be relative to the template directory.
func (t *templateBuilder) Directories(directories ...string) *templateBuilder {
	t.build.directories = directories
	return t
}

// Execute executes a template with the given data and returns the output
func (t *TemplateParsed) Execute(data interface{}) (*bytes.Buffer, error) {
	if t.Template == nil {
		return nil, errors.New("cannot execute template: template not initialized")
	}

	buf := new(bytes.Buffer)
	err := t.Template.ExecuteTemplate(buf, t.build.base+config.TemplateExt, data)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// Execute executes a template with the given data
// If the template has not been cached, it will be parsed and cached.
func (t *templateBuilder) Execute(data interface{}) (*bytes.Buffer, error) {
	tp, err := t.Store()
	if err != nil {
		return nil, err
	}

	return tp.Execute(data)
}

// Files sets the list of template files to include in the parse.
// This should not include the file extension and the files should be relative to the
// template directory.
func (t *templateBuilder) Files(files ...string) *templateBuilder {
	t.build.files = files
	return t
}

// Parse creates a template build operation.
func (t *TemplateRenderer) Parse() *templateBuilder {
	return &templateBuilder{
		renderer: t,
		build:    &templateBuild{},
	}
}

// Key sets the cache key for the template being built
func (t *templateBuilder) Key(key string) *templateBuilder {
	t.build.key = key
	return t
}

// Group sets the cache group for the template being built
func (t *templateBuilder) Group(group string) *templateBuilder {
	t.build.group = group
	return t
}

// Store parses the templates and stores them in the cache
func (t *templateBuilder) Store() (*TemplateParsed, error) {
	return t.renderer.parse(t.build)
}

// Load loads a template from cache
func (t *TemplateRenderer) Load(group, key string) (*TemplateParsed, error) {
	value, ok := t.templateCache.Load(t.getCacheKey(group, key))
	if !ok {
		return nil, errors.New("uncached page template requested")
	}

	if tmpl, ok := value.(*TemplateParsed); ok {
		return tmpl, nil
	}

	return nil, errors.New("unable to cast cached template")
}

// getCacheKey generates a cache key for a given group and key.
func (t *TemplateRenderer) getCacheKey(group, key string) string {
	if group != "" {
		return fmt.Sprintf("%s:%s", group, key)
	}
	return key
}

// parse parses a set of templates and caches them for quick execution. If the
// application environment is set to local, the cache will be bypassed and
// templates will be parsed upon each request so hot-reloading is possible
// without restarts. Also included will be the function map provided by the
// funcmap package.
func (t *TemplateRenderer) parse(build *templateBuild) (*TemplateParsed, error) {
	var tp *TemplateParsed
	var err error

	switch {
	case build.key == "":
		return nil, errors.New("cannot parse template without a cache key")
	case len(build.files) == 0 && len(build.directories) == 0:
		return nil, errors.New("cannot parse template without files or directories")
	case build.base == "":
		return nil, errors.New("cannot parse template without base")
	}

	// Generate the cache key
	cacheKey := t.getCacheKey(build.group, build.key)

	// Check if the template has not yet been parsed; or if the environment is development.
	// If the environment is local we want the templates to reflect changes without having
	// the server restart.
	devEnv := t.config.App.Environment == config.EnvDev || t.config.App.Environment == config.EnvLocal
	if tp, err = t.Load(build.group, build.key); err != nil || devEnv {
		// Initialize the parsed template with the function map
		parsed := template.New(build.base + config.TemplateExt).
			Funcs(t.funcMap).
			Funcs(template.FuncMap{"file": funcmap.File})

		// Parse all the provided files
		if len(build.files) > 0 {
			for k, v := range build.files {
				build.files[k] = fmt.Sprintf("%s/%s%s", t.templatesPath, v, config.TemplateExt)
			}

			parsed, err = parsed.ParseFiles(build.files...)
			if err != nil {
				return nil, err
			}
		}

		// Parse all templates within the provided directories
		for _, dir := range build.directories {
			dir = fmt.Sprintf("%s/%s/*%s", t.templatesPath, dir, config.TemplateExt)
			parsed, err = parsed.ParseGlob(dir)
			if err != nil {
				return nil, err
			}
		}

		// Store the template so this process only happens once
		tp = &TemplateParsed{
			Template: parsed,
			build:    build,
		}
		t.templateCache.Store(cacheKey, tp)
	}

	return tp, err
}
