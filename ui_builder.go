package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
)

//go:embed resources/ui/src resources/ui/src/**/*
var uiSources embed.FS

type UIBuilder struct {
	cache map[string]string
	mutex sync.RWMutex
}

func NewUIBuilder() *UIBuilder {
	return &UIBuilder{
		cache: make(map[string]string),
	}
}

// BuildReactBundle compiles the TypeScript React app with embedded media data
func (ub *UIBuilder) BuildReactBundle(mediaData interface{}) (string, error) {
	// Create cache key from media data
	dataJSON, err := json.Marshal(mediaData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal media data: %w", err)
	}

	hash := sha256.Sum256(dataJSON)
	cacheKey := hex.EncodeToString(hash[:])

	// Check cache first
	ub.mutex.RLock()
	if cached, exists := ub.cache[cacheKey]; exists {
		ub.mutex.RUnlock()
		slog.Debug("Using cached UI bundle", "cacheKey", cacheKey[:8])
		return cached, nil
	}
	ub.mutex.RUnlock()

	// Read all source files for esbuild first
	sourceFiles := make(map[string]string)
	err = ub.readSourceFiles("resources/ui/src", sourceFiles)
	if err != nil {
		return "", fmt.Errorf("failed to read source files: %w", err)
	}

	slog.Debug("Read embedded TypeScript files", "count", len(sourceFiles))
	for path := range sourceFiles {
		slog.Debug("Found embedded file", "path", path)
	}

	// Check if we have the main entry point
	indexContent, exists := sourceFiles["index.tsx"]
	if !exists {
		return "", fmt.Errorf("index.tsx not found in embedded sources (available: %v)", keys(sourceFiles))
	}

	// Inject media data as a global constant
	dataConstant := fmt.Sprintf(`
// Injected media data
const MEDIA_DATA = %s;

// Override the media data hook to use injected data
window.__MEDIA_DATA__ = MEDIA_DATA;

%s`, string(dataJSON), indexContent)

	// Update the index.tsx content with injected data
	sourceFiles["index.tsx"] = dataConstant

	// Build with esbuild using virtual file system
	result := api.Build(api.BuildOptions{
		Bundle:            true,
		Write:             false,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Format:            api.FormatIIFE,
		Target:            api.ES2015,
		Platform:          api.PlatformBrowser,
		JSX:               api.JSXAutomatic,
		GlobalName:        "MediaApp",
		EntryPoints:       []string{"virtual:index.tsx"},
		Plugins: []api.Plugin{
			{
				Name: "react-globals",
				Setup: func(build api.PluginBuild) {
					// Map React imports to global variables
					build.OnResolve(api.OnResolveOptions{Filter: `^react$`},
						func(args api.OnResolveArgs) (api.OnResolveResult, error) {
							return api.OnResolveResult{
								Path:      "react",
								Namespace: "react-globals",
							}, nil
						})
					build.OnResolve(api.OnResolveOptions{Filter: `^react-dom$`},
						func(args api.OnResolveArgs) (api.OnResolveResult, error) {
							return api.OnResolveResult{
								Path:      "react-dom",
								Namespace: "react-globals",
							}, nil
						})
					build.OnResolve(api.OnResolveOptions{Filter: `^react-dom/client$`},
						func(args api.OnResolveArgs) (api.OnResolveResult, error) {
							return api.OnResolveResult{
								Path:      "react-dom/client",
								Namespace: "react-globals",
							}, nil
						})
					build.OnResolve(api.OnResolveOptions{Filter: `^react/jsx-runtime$`},
						func(args api.OnResolveArgs) (api.OnResolveResult, error) {
							return api.OnResolveResult{
								Path:      "react/jsx-runtime",
								Namespace: "react-globals",
							}, nil
						})

					// Provide the global variable mappings
					build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "react-globals"},
						func(args api.OnLoadArgs) (api.OnLoadResult, error) {
							var contents string
							switch args.Path {
							case "react":
								contents = "module.exports = window.React;"
							case "react-dom":
								contents = "module.exports = window.ReactDOM;"
							case "react-dom/client":
								contents = "module.exports = window.ReactDOM;"
							case "react/jsx-runtime":
								contents = `
var React = window.React;
function jsx(type, props, key) {
  if (props && props.children !== undefined) {
    return React.createElement(type, key ? {...props, key: key} : props, props.children);
  }
  return React.createElement(type, key ? {...props, key: key} : props);
}
function jsxs(type, props, key) {
  if (props && props.children !== undefined) {
    return React.createElement(type, key ? {...props, key: key} : props, ...Array.isArray(props.children) ? props.children : [props.children]);
  }
  return React.createElement(type, key ? {...props, key: key} : props);
}
module.exports = { jsx: jsx, jsxs: jsxs, Fragment: React.Fragment };
`
							default:
								return api.OnLoadResult{}, fmt.Errorf("unknown react global: %s", args.Path)
							}
							return api.OnLoadResult{
								Contents: &contents,
								Loader:   api.LoaderJS,
							}, nil
						})
				},
			},
			{
				Name: "virtual-fs",
				Setup: func(build api.PluginBuild) {
					// Handle virtual: protocol for entry points
					build.OnResolve(api.OnResolveOptions{Filter: `^virtual:`},
						func(args api.OnResolveArgs) (api.OnResolveResult, error) {
							path := strings.TrimPrefix(args.Path, "virtual:")
							return api.OnResolveResult{
								Path:      path,
								Namespace: "virtual",
							}, nil
						})

					// Handle relative imports
					build.OnResolve(api.OnResolveOptions{Filter: `^\.`},
						func(args api.OnResolveArgs) (api.OnResolveResult, error) {
							path := args.Path
							importer := args.Importer

							// Resolve relative paths based on the importer's directory
							if strings.HasPrefix(path, "../") {
								// Go up one directory from importer
								importerDir := ""
								if strings.Contains(importer, "/") {
									parts := strings.Split(importer, "/")
									if len(parts) > 1 {
										importerDir = strings.Join(parts[:len(parts)-2], "/")
										if importerDir != "" {
											importerDir += "/"
										}
									}
								}
								path = importerDir + strings.TrimPrefix(path, "../")
							} else if strings.HasPrefix(path, "./") {
								// Same directory as importer
								importerDir := ""
								if strings.Contains(importer, "/") {
									parts := strings.Split(importer, "/")
									if len(parts) > 1 {
										importerDir = strings.Join(parts[:len(parts)-1], "/") + "/"
									}
								}
								path = importerDir + strings.TrimPrefix(path, "./")
							}

							// Try to find the file with or without extension
							if _, exists := sourceFiles[path]; exists {
								return api.OnResolveResult{
									Path:      path,
									Namespace: "virtual",
								}, nil
							}

							// Try with .tsx extension
							if _, exists := sourceFiles[path+".tsx"]; exists {
								return api.OnResolveResult{
									Path:      path + ".tsx",
									Namespace: "virtual",
								}, nil
							}

							// Try with .ts extension
							if _, exists := sourceFiles[path+".ts"]; exists {
								return api.OnResolveResult{
									Path:      path + ".ts",
									Namespace: "virtual",
								}, nil
							}

							return api.OnResolveResult{}, fmt.Errorf("virtual file not found: %s from %s -> resolved to %s (available: %v)", args.Path, importer, path, keys(sourceFiles))
						})

					// Load files from our virtual file system
					build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "virtual"},
						func(args api.OnLoadArgs) (api.OnLoadResult, error) {
							if content, exists := sourceFiles[args.Path]; exists {
								var loader api.Loader
								if strings.HasSuffix(args.Path, ".tsx") {
									loader = api.LoaderTSX
								} else {
									loader = api.LoaderTS
								}

								return api.OnLoadResult{
									Contents: &content,
									Loader:   loader,
								}, nil
							}
							return api.OnLoadResult{}, fmt.Errorf("virtual file not found: %s", args.Path)
						})
				},
			},
		},
	})

	if len(result.Errors) > 0 {
		var errorMessages []string
		for _, err := range result.Errors {
			errorMessages = append(errorMessages, err.Text)
		}
		return "", fmt.Errorf("esbuild errors: %s", strings.Join(errorMessages, "; "))
	}

	if len(result.OutputFiles) == 0 {
		return "", fmt.Errorf("no output files generated")
	}

	bundle := string(result.OutputFiles[0].Contents)

	// Cache the result
	ub.mutex.Lock()
	ub.cache[cacheKey] = bundle
	ub.mutex.Unlock()

	slog.Debug("Built and cached UI bundle",
		"cacheKey", cacheKey[:8],
		"bundleSize", len(bundle))

	return bundle, nil
}

func (ub *UIBuilder) readSourceFiles(basePath string, files map[string]string) error {
	entries, err := uiSources.ReadDir(basePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := basePath + "/" + entry.Name()

		if entry.IsDir() {
			err := ub.readSourceFiles(fullPath, files)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(entry.Name(), ".ts") || strings.HasSuffix(entry.Name(), ".tsx") {
			content, err := uiSources.ReadFile(fullPath)
			if err != nil {
				return err
			}

			// Convert full path to relative path for esbuild
			relativePath := strings.TrimPrefix(fullPath, "resources/ui/src/")
			files[relativePath] = string(content)
		}
	}

	return nil
}

func keys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
