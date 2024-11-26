// Code generated by go-enum DO NOT EDIT.
// Version: 0.6.0
// Revision: 919e61c0174b91303753ee3898569a01abb32c97
// Build Date: 2023-12-18T15:54:43Z
// Built By: goreleaser

package stdlib

import (
	"fmt"
	"strings"
)

const (
	// FakeStrategyUnspecified is a FakeStrategy of type unspecified.
	FakeStrategyUnspecified FakeStrategy = "unspecified"
	// FakeStrategyRandom is a FakeStrategy of type random.
	FakeStrategyRandom FakeStrategy = "random"
	// FakeStrategyRandomRange is a FakeStrategy of type random_range.
	FakeStrategyRandomRange FakeStrategy = "random_range"
	// FakeStrategyRandomPattern is a FakeStrategy of type random_pattern.
	FakeStrategyRandomPattern FakeStrategy = "random_pattern"
	// FakeStrategyRandomSelect is a FakeStrategy of type random_select.
	FakeStrategyRandomSelect FakeStrategy = "random_select"
	// FakeStrategyDistributionNormal is a FakeStrategy of type distribution_normal.
	FakeStrategyDistributionNormal FakeStrategy = "distribution_normal"
	// FakeStrategyDistributionUniform is a FakeStrategy of type distribution_uniform.
	FakeStrategyDistributionUniform FakeStrategy = "distribution_uniform"
	// FakeStrategyStateful is a FakeStrategy of type stateful.
	FakeStrategyStateful FakeStrategy = "stateful"
)

var ErrInvalidFakeStrategy = fmt.Errorf("not a valid FakeStrategy, try [%s]", strings.Join(_FakeStrategyNames, ", "))

var _FakeStrategyNames = []string{
	string(FakeStrategyUnspecified),
	string(FakeStrategyRandom),
	string(FakeStrategyRandomRange),
	string(FakeStrategyRandomPattern),
	string(FakeStrategyRandomSelect),
	string(FakeStrategyDistributionNormal),
	string(FakeStrategyDistributionUniform),
	string(FakeStrategyStateful),
}

// FakeStrategyNames returns a list of possible string values of FakeStrategy.
func FakeStrategyNames() []string {
	tmp := make([]string, len(_FakeStrategyNames))
	copy(tmp, _FakeStrategyNames)
	return tmp
}

// String implements the Stringer interface.
func (x FakeStrategy) String() string {
	return string(x)
}

// IsValid provides a quick way to determine if the typed value is
// part of the allowed enumerated values
func (x FakeStrategy) IsValid() bool {
	_, err := ParseFakeStrategy(string(x))
	return err == nil
}

var _FakeStrategyValue = map[string]FakeStrategy{
	"unspecified":          FakeStrategyUnspecified,
	"random":               FakeStrategyRandom,
	"random_range":         FakeStrategyRandomRange,
	"random_pattern":       FakeStrategyRandomPattern,
	"random_select":        FakeStrategyRandomSelect,
	"distribution_normal":  FakeStrategyDistributionNormal,
	"distribution_uniform": FakeStrategyDistributionUniform,
	"stateful":             FakeStrategyStateful,
}

// ParseFakeStrategy attempts to convert a string to a FakeStrategy.
func ParseFakeStrategy(name string) (FakeStrategy, error) {
	if x, ok := _FakeStrategyValue[name]; ok {
		return x, nil
	}
	return FakeStrategy(""), fmt.Errorf("%s is %w", name, ErrInvalidFakeStrategy)
}

// MarshalText implements the text marshaller method.
func (x FakeStrategy) MarshalText() ([]byte, error) {
	return []byte(string(x)), nil
}

// UnmarshalText implements the text unmarshaller method.
func (x *FakeStrategy) UnmarshalText(text []byte) error {
	tmp, err := ParseFakeStrategy(string(text))
	if err != nil {
		return err
	}
	*x = tmp
	return nil
}
