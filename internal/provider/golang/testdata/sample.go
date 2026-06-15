package sample

type SampleInterface interface {
	PublicMethod() int
}

type SampleStruct struct {
	Value int
}

type SampleType string

const (
	PublicConst  = "public"
	privateConst = "private"
)

var (
	PublicVar  = SampleStruct{}
	privateVar = SampleStruct{}
)

func PublicFunc(input int) int {
	if input > 0 {
		return input
	}

	return -input
}

func privateFunc(input int) int {
	return input * 2
}

func (s SampleStruct) PublicMethod() int {
	return s.Value
}

func (s SampleStruct) privateMethod(input int) int {
	if input > 0 && s.Value > 0 {
		return input + s.Value
	}

	return s.Value
}
