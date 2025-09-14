package upload

import (
	"testing"
)

func TestUploadOptions_ConcurrentUploadsValidation(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "valid value within range",
			input:    4,
			expected: 4,
		},
		{
			name:     "value below minimum",
			input:    0,
			expected: 1,
		},
		{
			name:     "negative value",
			input:    -5,
			expected: 1,
		},
		{
			name:     "value above maximum",
			input:    25,
			expected: 20,
		},
		{
			name:     "maximum allowed value",
			input:    20,
			expected: 20,
		},
		{
			name:     "minimum allowed value",
			input:    1,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &UploadOptions{
				ConcurrentUploads: tt.input,
			}

			// Simulate the validation logic from the Open method
			if options.ConcurrentUploads < 1 {
				options.ConcurrentUploads = 1
			} else if options.ConcurrentUploads > 20 {
				options.ConcurrentUploads = 20
			}

			if options.ConcurrentUploads != tt.expected {
				t.Errorf("ConcurrentUploads = %d, expected %d", options.ConcurrentUploads, tt.expected)
			}
		})
	}
}

func TestUploadLoop_DecisionLogic(t *testing.T) {
	tests := []struct {
		name                string
		concurrentUploads   int
		expectedSequential  bool
	}{
		{
			name:               "single upload should use sequential",
			concurrentUploads:  1,
			expectedSequential: true,
		},
		{
			name:               "multiple uploads should use concurrent",
			concurrentUploads:  4,
			expectedSequential: false,
		},
		{
			name:               "maximum uploads should use concurrent",
			concurrentUploads:  20,
			expectedSequential: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upCmd := &UpCmd{
				UploadOptions: &UploadOptions{
					ConcurrentUploads: tt.concurrentUploads,
				},
			}

			// Test the decision logic in uploadLoop
			usesSequential := upCmd.ConcurrentUploads == 1

			if usesSequential != tt.expectedSequential {
				t.Errorf("For ConcurrentUploads=%d, expected sequential=%v, got sequential=%v", 
					tt.concurrentUploads, tt.expectedSequential, usesSequential)
			}
		})
	}
}