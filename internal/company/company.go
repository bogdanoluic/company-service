package company

import "github.com/google/uuid"

type Type string

const (
	TypeCorporations       Type = "Corporations"
	TypeNonProfit          Type = "NonProfit"
	TypeCooperative        Type = "Cooperative"
	TypeSoleProprietorship Type = "Sole Proprietorship"
)

type Company struct {
	ID                uuid.UUID
	Name              string
	Description       *string
	AmountOfEmployees int
	Registered        bool
	Type              Type
}

func (t Type) Valid() bool {
	switch t {
	case TypeCorporations,
		TypeNonProfit,
		TypeCooperative,
		TypeSoleProprietorship:
		return true
	default:
		return false
	}
}
