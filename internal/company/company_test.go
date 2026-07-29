package company

import "testing"

func TestTypeValid(t *testing.T) {
	tests := []struct {
		name        string
		companyType Type
		want        bool
	}{
		{
			name:        "corporations",
			companyType: TypeCorporations,
			want:        true,
		},
		{
			name:        "non-profit",
			companyType: TypeNonProfit,
			want:        true,
		},
		{
			name:        "cooperative",
			companyType: TypeCooperative,
			want:        true,
		},
		{
			name:        "sole proprietorship",
			companyType: TypeSoleProprietorship,
			want:        true,
		},
		{
			name:        "unknown type",
			companyType: Type("Unknown"),
			want:        false,
		},
		{
			name:        "empty type",
			companyType: "",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.companyType.Valid()

			if got != tt.want {
				t.Errorf(
					"Type(%q).Valid() = %t, want %t",
					tt.companyType,
					got,
					tt.want,
				)
			}
		})
	}
}
