package mcp

import (
	"fmt"
	"os"
	"strings"
)

// ServerConfig represents configuration for an MCP server
type ServerConfig struct {
	// Name is a unique identifier for this server
	Name string `mapstructure:"name" json:"name" yaml:"name"`

	// TransportType is the transport type: "stdio", "http", or "sse"
	// Defaults to "stdio" if Command is set, "http" if URL is set
	TransportType string `mapstructure:"transport_type,omitempty" json:"transport_type,omitempty" yaml:"transport_type,omitempty"`

	// Command is the command to run (for stdio transport)
	Command string `mapstructure:"command,omitempty" json:"command,omitempty" yaml:"command,omitempty"`

	// Args are the command arguments
	Args []string `mapstructure:"args,omitempty" json:"args,omitempty" yaml:"args,omitempty"`

	// Env is a map of environment variables
	Env map[string]string `mapstructure:"env,omitempty" json:"env,omitempty" yaml:"env,omitempty"`

	// URL is the HTTP/SSE endpoint URL (for HTTP/SSE transport)
	URL string `mapstructure:"url,omitempty" json:"url,omitempty" yaml:"url,omitempty"`

	// Headers are HTTP headers for HTTP/SSE transport
	Headers map[string]string `mapstructure:"headers,omitempty" json:"headers,omitempty" yaml:"headers,omitempty"`

	// Disabled disables this server
	Disabled bool `mapstructure:"disabled,omitempty" json:"disabled,omitempty" yaml:"disabled,omitempty"`

	// Timeout for requests (default: 30s)
	Timeout string `mapstructure:"timeout,omitempty" json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// Config represents MCP configuration
type Config struct {
	// Servers is a list of MCP server configurations
	Servers []ServerConfig `mapstructure:"servers" json:"Servers" yaml:"servers"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	names := make(map[string]bool)

	for i, server := range c.Servers {
		if server.Name == "" {
			return fmt.Errorf("server %d: name is required", i)
		}

		if names[server.Name] {
			return fmt.Errorf("duplicate server name: %s", server.Name)
		}
		names[server.Name] = true

		// Validate transport configuration
		hasStdio := server.Command != ""
		hasURL := server.URL != ""
		transportType := server.TransportType

		// Auto-detect transport type if not specified
		if transportType == "" {
			if hasStdio && hasURL {
				return fmt.Errorf("server %s: cannot specify both command and url without explicit transport_type", server.Name)
			}
			if !hasStdio && !hasURL {
				return fmt.Errorf("server %s: either command or url must be specified", server.Name)
			}
		} else {
			// Validate explicit transport type
			switch transportType {
			case "stdio":
				if !hasStdio {
					return fmt.Errorf("server %s: transport_type 'stdio' requires command", server.Name)
				}
			case "http", "sse":
				if !hasURL {
					return fmt.Errorf("server %s: transport_type '%s' requires url", server.Name, transportType)
				}
			default:
				return fmt.Errorf("server %s: invalid transport_type '%s', must be 'stdio', 'http', or 'sse'", server.Name, transportType)
			}
		}

		// Expand environment variables in env values
		for k, v := range server.Env {
			c.Servers[i].Env[k] = expandEnvVars(v)
		}

		// Expand environment variables in headers
		for k, v := range server.Headers {
			c.Servers[i].Headers[k] = expandEnvVars(v)
		}
	}

	return nil
}

// expandEnvVars expands environment variables in a string
func expandEnvVars(s string) string {
	// Support both $VAR and ${VAR} syntax
	return os.ExpandEnv(s)
}

// GetServer returns a server configuration by name
func (c *Config) GetServer(name string) *ServerConfig {
	for _, server := range c.Servers {
		if server.Name == name {
			return &server
		}
	}
	return nil
}

// AddServer adds a server configuration
func (c *Config) AddServer(server ServerConfig) error {
	// Check for duplicate name
	for _, s := range c.Servers {
		if s.Name == server.Name {
			return fmt.Errorf("server %s already exists", server.Name)
		}
	}

	c.Servers = append(c.Servers, server)
	return nil
}

// RemoveServer removes a server configuration
func (c *Config) RemoveServer(name string) error {
	for i, server := range c.Servers {
		if server.Name == name {
			c.Servers = append(c.Servers[:i], c.Servers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("server %s not found", name)
}

// UpdateServer updates a server configuration
func (c *Config) UpdateServer(name string, server ServerConfig) error {
	for i, s := range c.Servers {
		if s.Name == name {
			c.Servers[i] = server
			return nil
		}
	}
	return fmt.Errorf("server %s not found", name)
}

// GetEnabledServers returns only enabled servers
func (c *Config) GetEnabledServers() []ServerConfig {
	var enabled []ServerConfig
	for _, server := range c.Servers {
		if !server.Disabled {
			enabled = append(enabled, server)
		}
	}
	return enabled
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Servers: []ServerConfig{},
	}
}

// Example configurations for common MCP servers

// ExampleFilesystemConfig returns an example filesystem server config
func ExampleFilesystemConfig() ServerConfig {
	return ServerConfig{
		Name:    "filesystem",
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/files"},
	}
}

// ExampleGitHubConfig returns an example GitHub server config
func ExampleGitHubConfig() ServerConfig {
	return ServerConfig{
		Name:    "github",
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-github"},
		Env: map[string]string{
			"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}",
		},
	}
}

// ExampleFetchConfig returns an example fetch server config
func ExampleFetchConfig() ServerConfig {
	return ServerConfig{
		Name:    "fetch",
		Command: "uvx",
		Args:    []string{"mcp-server-fetch"},
	}
}

// ParseServerFromURL parses a simple server config from a URL (for quick add)
func ParseServerFromURL(name, urlStr string) (ServerConfig, error) {
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return ServerConfig{}, fmt.Errorf("URL must start with http:// or https://")
	}

	return ServerConfig{
		Name: name,
		URL:  urlStr,
	}, nil
}
