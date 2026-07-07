package pristinecss

import (
	"context"
	"fmt"
)

type CSSClass struct {
	Path  string
	Class string
}

type CSSLoader struct {
	cssFiles    map[string]string
	cssClasses  map[string]CSSClass
	alwaysLoad  []string
	loadedFiles map[string][]string
}

func NewLoader(cssFiles map[string]string, cssClasses map[string]CSSClass) *CSSLoader {
	return &CSSLoader{
		cssFiles:    cssFiles,
		cssClasses:  cssClasses,
		alwaysLoad:  []string{},
		loadedFiles: make(map[string][]string),
	}
}

// Class retrieves the CSS class name for a given class path and registers the file in the loading list if a context ID is set.
// The class path is in the format <filepath>.<classname> (dot-separated).
//
// The CSS class name returned is the one to be used in code. This can be a modified class name when using PostCSS modules.
//
// Parameters:
// - ctx: The context for the request, used to manage the class files that are used for the current context.
// - name: The class path of the CSS class to retrieve.
//
// Returns:
// - The CSS class name as a string.
// - An error if the class name is not found.
func (c *CSSLoader) Class(ctx context.Context, name string) (string, error) {
	class, ok := c.cssClasses[name]
	if !ok {
		return "", fmt.Errorf("class %s not found", name)
	}

	cssContextId, ok := fromContext(ctx)
	if ok {
		// Adds to list of files that we need to load for the current context
		c.registerFileForLoading(cssContextId, class)
	}

	return class.Class, nil
}

// registerFileForLoading registers a CSS file for loading in the given context.
// It ensures that the file is added only if it is not already present in the list.
//
// Parameters:
// - contextId: A unique identifier for the context.
// - class: The CSSClass instance containing the file path and class name.
func (c *CSSLoader) registerFileForLoading(contextId string, class CSSClass) {
	loadedFiles, ok := c.loadedFiles[contextId]
	if !ok {
		loadedFiles = append([]string{}, c.alwaysLoad...)
	}

	path, _ := c.GetPath(class.Path)

	for _, val := range loadedFiles {
		if path == val {
			return
		}
	}

	loadedFiles = append(loadedFiles, path)
	c.loadedFiles[contextId] = loadedFiles
}

// GetPath retrieves the versioned file path for a given CSS file path.
//
// Parameters:
// - name: The filepath of the CSS file.
//
// Returns:
// - The versioned file path as a string.
// - An error if the path is not found.
func (c *CSSLoader) GetPath(name string) (string, error) {
	path, ok := c.cssFiles[name]
	if !ok {
		return "", fmt.Errorf("path %s not found", name)
	}

	return path, nil
}

// LoadedFiles retrieves the list of CSS files loaded for the given context and clears the record.
// This should be called at the last moment as it clears the loaded files for the current context.
//
// Parameters:
// - ctx: The context for the request, used to manage the lifecycle of the request.
//
// Returns:
// - A slice of strings containing the paths of the loaded CSS files.
func (c *CSSLoader) LoadedFiles(ctx context.Context) []string {
	cssContextId, ok := fromContext(ctx)
	if !ok {
		return make([]string, 0)
	}
	files, ok := c.loadedFiles[cssContextId]
	if !ok {
		return make([]string, 0)
	}

	delete(c.loadedFiles, cssContextId)

	return files
}
