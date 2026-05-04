package validator

import "testing"

type gradeRequest struct {
	Class       string `validate:"checkValidGrade"`
	ClassUpdate string `validate:"checkValidGradeUpdate"`
}

func TestValidate(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		input   gradeRequest
		wantErr bool
	}{
		{
			name: "valid grade on create",
			input: gradeRequest{
				Class: "Grade 5",
			},
			wantErr: false,
		},
		{
			name: "invalid grade on create",
			input: gradeRequest{
				Class: "Grade 13",
			},
			wantErr: true,
		},
		{
			name: "valid grade on update",
			input: gradeRequest{
				ClassUpdate: "Grade 12",
			},
			wantErr: false,
		},
		{
			name: "invalid grade on update",
			input: gradeRequest{
				ClassUpdate: "Grade 0",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.Validate(&tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				if _, ok := err.(*ValidationError); !ok {
					t.Fatalf("expected *ValidationError, got %T", err)
				}
			}
		})
	}
}
