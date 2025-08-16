package api

import (
	"testing"
)

func TestValidateAndCleanPath_SecurityTests(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		desc      string
	}{
		// Valid paths
		{
			name:      "valid_index",
			input:     "/index.html",
			expectErr: false,
			desc:      "Valid index.html path",
		},
		{
			name:      "valid_css",
			input:     "/styles/main.css",
			expectErr: false,
			desc:      "Valid CSS file path",
		},
		{
			name:      "valid_js",
			input:     "/scripts/app.js",
			expectErr: false,
			desc:      "Valid JavaScript file path",
		},
		{
			name:      "valid_root",
			input:     "/",
			expectErr: false,
			desc:      "Valid root path",
		},
		
		// Directory traversal attempts
		{
			name:      "traversal_dotdot",
			input:     "/../etc/passwd",
			expectErr: true,
			desc:      "Directory traversal with ../",
		},
		{
			name:      "traversal_encoded",
			input:     "/%2e%2e/etc/passwd",
			expectErr: false, // URL decoding should happen before this function
			desc:      "URL encoded directory traversal",
		},
		{
			name:      "traversal_multiple",
			input:     "/../../etc/passwd",
			expectErr: true,
			desc:      "Multiple directory traversal attempts",
		},
		{
			name:      "traversal_middle",
			input:     "/static/../../../etc/passwd",
			expectErr: true,
			desc:      "Directory traversal in middle of path",
		},
		{
			name:      "traversal_end",
			input:     "/static/..",
			expectErr: true,
			desc:      "Directory traversal at end",
		},
		{
			name:      "traversal_backslash",
			input:     "/static\\..\\..\\etc\\passwd",
			expectErr: true,
			desc:      "Backslash directory traversal (Windows style)",
		},
		
		// Null byte injection
		{
			name:      "null_byte",
			input:     "/index.html\x00.jpg",
			expectErr: true,
			desc:      "Null byte injection attack",
		},
		
		// Hidden file access
		{
			name:      "hidden_file",
			input:     "/.env",
			expectErr: true,
			desc:      "Access to hidden .env file",
		},
		{
			name:      "hidden_ssh",
			input:     "/.ssh/id_rsa",
			expectErr: true,
			desc:      "Access to hidden SSH keys",
		},
		{
			name:      "hidden_git",
			input:     "/.git/config",
			expectErr: true,
			desc:      "Access to .git directory",
		},
		
		// File extension attacks
		{
			name:      "executable_file",
			input:     "/script.sh",
			expectErr: true,
			desc:      "Executable shell script",
		},
		{
			name:      "config_file",
			input:     "/config.ini",
			expectErr: true,
			desc:      "Configuration file with disallowed extension",
		},
		{
			name:      "backup_file",
			input:     "/index.html.bak",
			expectErr: true,
			desc:      "Backup file access",
		},
		
		// Edge cases
		{
			name:      "empty_path",
			input:     "",
			expectErr: false,
			desc:      "Empty path should be allowed for SPA routing",
		},
		{
			name:      "just_dot",
			input:     "/.",
			expectErr: false,
			desc:      "Single dot should be allowed",
		},
		{
			name:      "case_insensitive",
			input:     "/INDEX.HTML",
			expectErr: false,
			desc:      "Case insensitive file extensions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateAndCleanPath(tt.input)
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error for %s (%s), but got none. Result: %s", tt.input, tt.desc, result)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s (%s), but got: %v", tt.input, tt.desc, err)
				}
			}
		})
	}
}

func TestIsAllowedFileExtension_SecurityTests(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
		desc     string
	}{
		// Allowed extensions
		{"html", "index.html", true, "HTML files should be allowed"},
		{"css", "style.css", true, "CSS files should be allowed"},
		{"js", "app.js", true, "JavaScript files should be allowed"},
		{"json", "data.json", true, "JSON files should be allowed"},
		{"png", "image.png", true, "PNG images should be allowed"},
		{"svg", "icon.svg", true, "SVG images should be allowed"},
		{"ico", "favicon.ico", true, "ICO files should be allowed"},
		{"woff", "font.woff", true, "WOFF fonts should be allowed"},
		{"no_extension", "robots", true, "Files without extension should be allowed for SPA routing"},
		
		// Case insensitivity
		{"html_upper", "index.HTML", true, "Uppercase extensions should work"},
		{"css_mixed", "style.Css", true, "Mixed case extensions should work"},
		
		// Disallowed extensions
		{"php", "script.php", false, "PHP files should be blocked"},
		{"sh", "script.sh", false, "Shell scripts should be blocked"},
		{"exe", "malware.exe", false, "Executables should be blocked"},
		{"bat", "script.bat", false, "Batch files should be blocked"},
		{"config", "app.config", false, "Config files should be blocked"},
		{"env", "database.env", false, "Environment files should be blocked"},
		{"key", "private.key", false, "Key files should be blocked"},
		{"pem", "cert.pem", false, "Certificate files should be blocked"},
		{"log", "access.log", false, "Log files should be blocked"},
		{"bak", "index.html.bak", false, "Backup files should be blocked"},
		{"tmp", "temp.tmp", false, "Temporary files should be blocked"},
		
		// Double extensions
		{"double_ext", "file.php.txt", false, "Files with disallowed extension anywhere should be blocked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAllowedFileExtension(tt.path)
			
			if result != tt.expected {
				t.Errorf("isAllowedFileExtension(%s) = %v, expected %v (%s)", 
					tt.path, result, tt.expected, tt.desc)
			}
		})
	}
}