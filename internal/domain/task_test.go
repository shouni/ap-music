package domain

import "testing"

func TestTaskValidateSubmission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		task    Task
		wantErr bool
	}{
		{name: "url only", task: Task{RequestURL: "https://example.com"}},
		{name: "text only", task: Task{InputText: "hello"}},
		{name: "image only", task: Task{ImageURL: "https://example.com/image.png"}},
		{name: "all empty", task: Task{}, wantErr: true},
		{name: "whitespace only", task: Task{RequestURL: " ", InputText: "\n", ImageURL: "\t"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.task.ValidateSubmission()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
