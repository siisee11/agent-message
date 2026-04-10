package cmd

import "testing"

func TestBuildSchemaDocumentCoversFullCommandTree(t *testing.T) {
	t.Parallel()

	document, err := buildSchemaDocument(NewRootCommand(), nil)
	if err != nil {
		t.Fatalf("buildSchemaDocument: %v", err)
	}

	if got, want := len(document.Commands), 34; got != want {
		t.Fatalf("command count mismatch: got %d want %d", got, want)
	}
	assertHasCommandPath(t, document, "agent-message")
	assertHasCommandPath(t, document, "agent-message schema")
	assertHasCommandPath(t, document, "agent-message send")
	assertHasCommandPath(t, document, "agent-message config set")
	assertHasCommandPath(t, document, "agent-message watch")
	assertHasCommandPath(t, document, "agent-message username")
	assertHasCommandPath(t, document, "agent-message username set")
	assertHasCommandPath(t, document, "agent-message username clear")
	assertHasCommandPath(t, document, "agent-message title")
	assertHasCommandPath(t, document, "agent-message title set")
	assertHasCommandPath(t, document, "agent-message title clear")
}

func TestBuildSchemaDocumentForSendIncludesTypedModes(t *testing.T) {
	t.Parallel()

	document, err := buildSchemaDocument(NewRootCommand(), []string{"send"})
	if err != nil {
		t.Fatalf("buildSchemaDocument(send): %v", err)
	}
	if got, want := len(document.Commands), 1; got != want {
		t.Fatalf("command count mismatch: got %d want %d", got, want)
	}

	send := document.Commands[0]
	if got, want := send.Path, "agent-message send"; got != want {
		t.Fatalf("path mismatch: got %q want %q", got, want)
	}
	if got, want := len(send.InputModes), 4; got != want {
		t.Fatalf("input mode count mismatch: got %d want %d", got, want)
	}

	kindFlag := findFlag(send.LocalFlags, "kind")
	if kindFlag == nil {
		t.Fatalf("expected kind flag in local flags")
	}
	if got, want := len(kindFlag.Enum), 2; got != want {
		t.Fatalf("kind enum length mismatch: got %d want %d", got, want)
	}
	if kindFlag.Enum[0] != "text" || kindFlag.Enum[1] != "json_render" {
		t.Fatalf("unexpected kind enum values: %+v", kindFlag.Enum)
	}

	foundReference := false
	for _, mode := range send.InputModes {
		for _, reference := range mode.References {
			if reference.Command == "agent-message catalog prompt" {
				foundReference = true
			}
		}
	}
	if !foundReference {
		t.Fatalf("expected send schema to reference agent-message catalog prompt")
	}
	if findFlag(send.LocalFlags, "payload") == nil {
		t.Fatalf("expected payload flag in send schema")
	}
}

func TestBuildSchemaDocumentForSchemaCommandIsSelfDescribing(t *testing.T) {
	t.Parallel()

	document, err := buildSchemaDocument(NewRootCommand(), []string{"schema"})
	if err != nil {
		t.Fatalf("buildSchemaDocument(schema): %v", err)
	}
	if got, want := len(document.Commands), 1; got != want {
		t.Fatalf("command count mismatch: got %d want %d", got, want)
	}

	schema := document.Commands[0]
	if !schema.SupportsJSONHelp || !schema.SupportsJSONValue {
		t.Fatalf("schema command should advertise JSON help and JSON value support: %+v", schema)
	}
	if got, want := len(schema.PositionalArgs), 1; got != want {
		t.Fatalf("schema positional arg count mismatch: got %d want %d", got, want)
	}
	if got, want := schema.PositionalArgs[0].Name, "command-path"; got != want {
		t.Fatalf("schema arg name mismatch: got %q want %q", got, want)
	}
	if !schema.PositionalArgs[0].Variadic {
		t.Fatalf("schema command-path arg should be variadic")
	}
	if got, want := schema.OutputFormats[0], "json"; got != want {
		t.Fatalf("schema output format mismatch: got %q want %q", got, want)
	}
}

func assertHasCommandPath(t *testing.T, document schemaDocument, want string) {
	t.Helper()

	for _, command := range document.Commands {
		if command.Path == want {
			return
		}
	}
	t.Fatalf("expected command path %q in schema document", want)
}

func findFlag(flags []schemaFlag, name string) *schemaFlag {
	for idx := range flags {
		if flags[idx].Name == name {
			return &flags[idx]
		}
	}
	return nil
}
