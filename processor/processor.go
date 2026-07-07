package processor

import (
	"io"
	"log"
	"os"
)

type ProcessorOpt func(*processorOptions)

type processorOptions struct {
	verbose   bool
	fileType  string
	outputDir string
	stylesDir string
}

func WithVerbose(verbose bool) ProcessorOpt {
	return func(opts *processorOptions) {
		opts.verbose = verbose
	}
}

func WithCssType() ProcessorOpt {
	return func(opts *processorOptions) {
		opts.fileType = ".css"
	}
}

// WithOutputDir sets the directory where the .go files are generated
func WithOutputDir(outputDir string) ProcessorOpt {
	return func(opts *processorOptions) {
		opts.outputDir = outputDir
	}
}

// WithStylesDir sets the directory specifically for CSS files
func WithStylesDir(stylesDir string) ProcessorOpt {
	return func(opts *processorOptions) {
		opts.stylesDir = stylesDir
	}
}

type ProcessFunction func(reader io.Reader) ProcessFunction

func Process(reader io.Reader, opts ...ProcessorOpt) {
	options := &processorOptions{
		fileType:  ".css",
		outputDir: "./public",
		stylesDir: "./public/styles",
	}

	for _, opt := range opts {
		opt(options)
	}

	err := os.MkdirAll(options.stylesDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create directory: %s, error: %v", options.stylesDir, err)
		return
	}

	// processing logic here
	// process the file with processor (css, scss, tailwind, postcss)
	// take a hash of the files contents
	// add to the this to a list of relative filepath => hash
	// add list of available css classes to classpath => CSSClass
	// write modified content to reletve path in output folder

	log.Printf("Finished processing files in directory: %s", source)
}

// createDirIfNotExists creates a directory only if it does not already exist.
func createDirIfNotExists(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err := os.Mkdir(dirPath, 0755)
		if err != nil {
			return err
		}
		log.Printf("Directory created: %s", dirPath)
	} else if err != nil {
		return err
	} else {
		log.Printf("Directory already exists: %s", dirPath)
	}
	return nil
}
