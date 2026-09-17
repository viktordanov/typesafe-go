package typesafe

// Questions maps names to questions. Answers use the same names.
type Questions map[string]Question

// Question is a Noul, Choice, or Score.
type Question interface{ isQuestion() }

// Noul asks a yes/no question. Its answer is the probability of yes.
//
// Instructions and descriptions accept text or any JSON-encodable value.
type Noul struct {
	Instructions any
	Criteria     *NoulCriteria
}

type NoulCriteria struct {
	True  any `json:"true,omitempty"`
	False any `json:"false,omitempty"`
}

// Choice selects one of 1 to 255 labels. A nil description is allowed.
type Choice struct {
	Instructions any
	Criteria     map[string]any
}

// Score rates against 2 to 10 ordered level descriptions, scored from zero.
type Score struct {
	Instructions any
	Criteria     []any
}

func (Noul) isQuestion()   {}
func (Choice) isQuestion() {}
func (Score) isQuestion()  {}
