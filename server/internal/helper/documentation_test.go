package helper

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/swaggo/swag"
	_ "github.com/zoff-music/vibes-backend/swaggerdocs"
)

type documentationFilterTest struct {
	name            string
	document        string
	wantError       bool
	wantPaths       int
	wantDefinitions int
}

func TestPublicDocumentation(t *testing.T) {
	generated, err := swag.ReadDoc()
	if err != nil {
		t.Fatal(err)
	}

	tests := []documentationFilterTest{
		{
			name:            "generated specification",
			document:        generated,
			wantPaths:       -1,
			wantDefinitions: -1,
		},
		{
			name: "tagged operation and transitive cyclic schemas",
			document: `{
  "swagger": "2.0",
  "paths": {
    "/public": {
      "get": {
        "responses": {
          "200": {
            "schema": {
              "$ref": "#/definitions/Public"
            }
          }
        }
      },
      "post": {
        "tags": [
          "admin"
        ]
      }
    },
    "/api/v2/admin/users": {
      "get": {}
    },
    "/only-admin": {
      "parameters": [],
      "get": {
        "tags": [
          "admin"
        ]
      }
    }
  },
  "definitions": {
    "Public": {
      "properties": {
        "child": {
          "$ref": "#/definitions/Child"
        }
      }
    },
    "Child": {
      "items": {
        "$ref": "#/definitions/Public"
      }
    },
    "Secret": {
      "type": "string"
    }
  },
  "tags": [
    {
      "name": "admin"
    }
  ]
}`,
			wantPaths:       1,
			wantDefinitions: 2,
		},
		{
			name: "no definitions",
			document: `{
  "swagger": "2.0",
  "paths": {
    "/public": {
      "get": {}
    }
  }
}`,
			wantPaths:       1,
			wantDefinitions: 0,
		},
		{
			name: "missing schema fails closed",
			document: `{
  "paths": {
    "/public": {
      "get": {
        "$ref": "#/definitions/Missing"
      }
    }
  }
}`,
			wantError: true,
		},
		{
			name:      "malformed input fails closed",
			document:  `{`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := PublicDocumentation(tt.document)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			var spec map[string]jsontext.Value
			err = json.Unmarshal(body, &spec)
			if err != nil {
				t.Fatal(err)
			}

			var paths map[string]jsontext.Value
			err = json.Unmarshal(spec["paths"], &paths)
			if err != nil {
				t.Fatal(err)
			}

			var definitions map[string]jsontext.Value
			err = json.Unmarshal(spec["definitions"], &definitions)
			if err != nil {
				t.Fatal(err)
			}

			if tt.wantPaths >= 0 && len(paths) != tt.wantPaths {
				t.Fatalf("paths %d, want %d", len(paths), tt.wantPaths)
			}

			if tt.wantDefinitions >= 0 && len(definitions) != tt.wantDefinitions {
				t.Fatalf("definitions %d, want %d", len(definitions), tt.wantDefinitions)
			}

			for path := range paths {
				if strings.Contains(path, "/admin/") || path == "/only-admin" {
					t.Fatalf("admin path exposed: %s", path)
				}
			}

			for name := range definitions {
				if strings.HasPrefix(name, "vibe.Admin") || name == "Secret" {
					t.Fatalf("admin schema exposed: %s", name)
				}
			}

			if strings.Contains(string(body), `"admin"`) {
				t.Fatal("admin tag exposed")
			}

			// Ensure every reference still resolves after pruning, including recursive models.
			refs, err := documentationReferences(body)
			if err != nil {
				t.Fatal(err)
			}

			for _, ref := range refs {
				if strings.HasPrefix(ref, "#/definitions/") {
					if _, ok := definitions[strings.TrimPrefix(ref, "#/definitions/")]; !ok {
						t.Fatalf("unresolved reference: %s", ref)
					}
				}
			}
		})
	}
}
