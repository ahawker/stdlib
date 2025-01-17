package stdlib

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"time"
)

const (
	// ErrorNamespaceDefault is the default namespace for errors generated by this package.
	ErrorNamespaceDefault = "stdlibx-go"
	// ErrorFlagUnknown is set to represent unknown/unregistered errors.
	ErrorFlagUnknown Bitmask = 1 << iota
	// ErrorFlagRetryable is set to represent errors that can be retried.
	ErrorFlagRetryable
	// ErrorFlagTimeout is set to represent errors indicating a timeout occurred.
	ErrorFlagTimeout
)

// ErrUndefined indicates the wrapped error is not well-known or previously
// defined. This likely means it's coming from an external system/library and not
// a domain error.
var ErrUndefined = Error{
	Code:      "undefined",
	Flags:     ErrorFlagUnknown,
	Message:   "wrapped the following error which is not well-defined",
	Namespace: ErrorNamespaceDefault,
}

// HasAs defines types necessary for stdlib `errors.As` support.
type HasAs interface {
	As(target any) bool
}

// HasIs defines types necessary for stdlib `errors.Is` support.
type HasIs interface {
	Is(target error) bool
}

// HasUnwrap defines types necessary for stdlib `errors.Unwrap` support.
type HasUnwrap interface {
	Unwrap() error
}

// Causer defines types that return the underlying cause of an error.
type Causer interface {
	Cause() error
}

// ErrorKey returns a slug that should be unique for each error (namespace + code).
func ErrorKey(namespace, code string) string {
	return fmt.Sprintf("%s/%s", namespace, code)
}

// Error defines a standard application error primitive.
//
// TODO(ahawker) Add Format interface (for pretty strings)
// TODO(ahawker) Namespace field? Embed in the code?
type Error struct {
	// Code is a machine-readable representation for the error.
	Code string `json:"code"`
	// Extras is an optional struct to store execution context
	// that is helpful for understanding the error.
	Extras ErrorExtras `json:"extras,omitempty"`
	// Flags is a bitmask that contains additional classification/context
	// for the error, e.g. indicating if the error can be retried.
	Flags Bitmask `json:"flags,omitempty"`
	// Message is a human-readable representation for the error.
	Message string `json:"message"`
	// Namespace is a machine-readable representation for a bucketing/grouping
	// concept of errors. This is commonly used to indicate the package/repository/service
	// an error originated from.
	Namespace string `json:"namespace"`
	// Wrapped is a wrapped error if this was created from another via `Wrap`. This
	// is hidden from human consumers and only visible to machine/operators.
	Wrapped error `json:"-"`
}

// Key returns a value that uniquely identifies the type of error.
func (e Error) Key() string {
	return ErrorKey(e.Namespace, e.Code)
}

// Equal returns true if the two Error values are equal.
func (e Error) Equal(e2 Error) bool {
	return e.Code == e2.Code &&
		e.Message == e2.Message &&
		e.Namespace == e2.Namespace &&
		e.Flags == e2.Flags &&
		reflect.DeepEqual(e.Extras.Debug, e2.Extras.Debug) &&
		reflect.DeepEqual(e.Extras.Help, e2.Extras.Help) &&
		reflect.DeepEqual(e.Extras.Retry, e2.Extras.Retry)
}

// IsZero returns true if the Error is an empty/zero value.
func (e Error) IsZero() bool {
	return reflect.DeepEqual(e, new(Error))
}

// IsRetryable returns true if the error indicates the failed operation
// is safe to retry.
func (e Error) IsRetryable() bool { return e.Flags.Has(ErrorFlagRetryable) }

// IsTimeout returns true if the error indicates an operation timeout.
func (e Error) IsTimeout() bool { return e.Flags.Has(ErrorFlagTimeout) }

// IsTransient returns true if the error indicates the operation failure
// is transient and a result might be different if tried at another time.
func (e Error) IsTransient() bool { return e.Flags.Has(ErrorFlagUnknown) }

// WithFlag returns a new copy of the Error with the given attribute applied.
func (e Error) WithFlag(attribute Bitmask) Error {
	return Error{
		Code:      e.Code,
		Extras:    e.Extras,
		Flags:     e.Flags.Set(attribute),
		Message:   e.Message,
		Namespace: e.Namespace,
		Wrapped:   e.Wrapped,
	}
}

// WithDebugInfo returns a new copy of the Error with the given debug info added.
func (e Error) WithDebugInfo(extras DebugExtras) Error {
	return Error{
		Code:      e.Code,
		Extras:    e.Extras.WithDebugExtras(extras),
		Flags:     e.Flags,
		Message:   e.Message,
		Namespace: e.Namespace,
		Wrapped:   e.Wrapped,
	}
}

// WithHelp returns a new copy of the Error with the given help info added.
func (e Error) WithHelp(extras HelpExtras) Error {
	return Error{
		Code:      e.Code,
		Extras:    e.Extras.WithHelpExtras(extras),
		Flags:     e.Flags,
		Message:   e.Message,
		Namespace: e.Namespace,
		Wrapped:   e.Wrapped,
	}
}

// WithRetry returns a new copy of the Error with the given retry info added.
func (e Error) WithRetry(extras RetryExtras) Error {
	return Error{
		Code:      e.Code,
		Extras:    e.Extras.WithRetryExtras(extras),
		Flags:     e.Flags,
		Message:   e.Message,
		Namespace: e.Namespace,
		Wrapped:   e.Wrapped,
	}
}

// WithTag returns a new copy of the Error with the given tags added.
func (e Error) WithTag(tags ...string) Error {
	return Error{
		Code:      e.Code,
		Extras:    e.Extras.WithTag(tags...),
		Flags:     e.Flags,
		Message:   e.Message,
		Namespace: e.Namespace,
		Wrapped:   e.Wrapped,
	}
}

// AsGroup returns a *ErrorGroup containing this error and all
// wrapped errors it contains.
func (e Error) AsGroup() *ErrorGroup {
	g := NewErrorGroup(e)

	err := e
	for err.Wrapped != nil {
		g.Append(err.Wrapped)

		var we Error
		if !errors.As(err.Wrapped, &we) {
			break
		}
		err = we
	}

	return g
}

// String returns the Error string representation.
//
// Interface: fmt.Stringer.
func (e Error) String() string {
	return e.Error()
}

// Format returns a complex string representation of the Error
// for the given verbs.
//
// Interface: fmt.Formatter.
func (e Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			if _, err := io.WriteString(s, e.AsGroup().Error()); err != nil {
				panic(err)
			}
			return
		}
		fallthrough
	case 's':
		if _, err := io.WriteString(s, e.Error()); err != nil {
			panic(err)
		}
	case 'q':
		if _, err := io.WriteString(s, e.Error()); err != nil {
			panic(err)
		}
	}
}

// Error returns the string representation of the Error.
//
// Interface: error.
func (e Error) Error() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s:%s] %s", e.Namespace, e.Code, e.Message))
	if e.Wrapped != nil {
		sb.WriteString(fmt.Sprintf("\n-> %s", e.Wrapped.Error()))
	}
	return sb.String()
}

// Is implements error equality checking.
//
// Interface: HasIs.
func (e Error) Is(target error) bool {
	var err Error
	if !errors.As(target, &err) {
		return false
	}
	return e.Equal(err)
}

// Unwrap implements error unwrapping for nested errors.
//
// Interface: Unwrap.
func (e Error) Unwrap() error {
	return e.Wrapped
}

// Wrap returns a new Error with the given err wrapped.
//
// If the given err is also an Error and the current instance
// is a zero value, just return a copy of the given Error. This
// allows us to avoid checking this case at every call-site; we
// can just Wrap the error and handle it.
func (e Error) Wrap(err error) Error {
	if err == nil {
		return e
	}
	if e.IsZero() {
		var e2 Error
		if errors.As(err, &e2) {
			return e2.Copy()
		}
	}
	return Error{
		Code:      e.Code,
		Extras:    e.Extras,
		Flags:     e.Flags,
		Message:   e.Message,
		Namespace: e.Namespace,
		Wrapped:   err,
	}
}

// Wrapf returns a new Error with an error created by the given format + args.
func (e Error) Wrapf(format string, a ...any) Error {
	return e.Wrap(fmt.Errorf(format, a...))
}

// Copy returns a full copy of this Error, including copies
// of all wrapped errors within.
func (e Error) Copy() Error {
	if e.Wrapped != nil {
		var wrapped Error

		if errors.As(e.Wrapped, &wrapped) {
			return Error{
				Code:      e.Code,
				Extras:    e.Extras,
				Flags:     e.Flags,
				Message:   e.Message,
				Namespace: e.Namespace,
				Wrapped:   wrapped.Copy(),
			}
		}
	}
	return Error{
		Code:      e.Code,
		Extras:    e.Extras,
		Flags:     e.Flags,
		Message:   e.Message,
		Namespace: e.Namespace,
		Wrapped:   e.Wrapped,
	}
}

var (
	_ Zeroer = (*ErrorExtras)(nil)
	_ Zeroer = (*DebugExtras)(nil)
	_ Zeroer = (*HelpExtras)(nil)
	_ Zeroer = (*RetryExtras)(nil)
)

// ErrorExtras contains common additional info attached to errors.
type ErrorExtras struct {
	// Debug information captured from the error.
	Debug DebugExtras `json:"debug,omitempty"`
	// Help information to inform operators about the error.
	Help HelpExtras `json:"help,omitempty"`
	// Retry information regarding the failed operation.
	Retry RetryExtras `json:"retry,omitempty"`
	// Tags are additional labels that can be used to categorize errors.
	Tags []string `json:"tags,omitempty"`
}

// WithDebugExtras returns a new copy of the ErrorExtras with the given debug info set.
func (e ErrorExtras) WithDebugExtras(extras DebugExtras) ErrorExtras {
	return ErrorExtras{
		Debug: extras,
		Help:  e.Help,
		Retry: e.Retry,
		Tags:  e.Tags,
	}
}

// WithHelpExtras returns a new copy of the ErrorExtras with the given help info set.
func (e ErrorExtras) WithHelpExtras(extras HelpExtras) ErrorExtras {
	return ErrorExtras{
		Debug: e.Debug,
		Help:  extras,
		Retry: e.Retry,
		Tags:  e.Tags,
	}
}

// WithRetryExtras returns a new copy of the ErrorExtras with the given retry info set.
func (e ErrorExtras) WithRetryExtras(extras RetryExtras) ErrorExtras {
	return ErrorExtras{
		Debug: e.Debug,
		Help:  e.Help,
		Retry: extras,
		Tags:  e.Tags,
	}
}

// WithTag returns a new copy of the ErrorExtras with the given tags set.
func (e ErrorExtras) WithTag(tags ...string) ErrorExtras {
	return ErrorExtras{
		Debug: e.Debug,
		Help:  e.Help,
		Retry: e.Retry,
		Tags:  append(e.Tags, tags...),
	}
}

// IsZero returns true if the ErrorExtras object is the zero/empty struct value.
func (e ErrorExtras) IsZero() bool {
	return e.Debug.IsZero() && e.Help.IsZero() && e.Retry.IsZero() && len(e.Tags) == 0
}

// DebugExtras contains helpful information for debugging the error.
type DebugExtras struct {
	// StackTrace of the error.
	StackTrace string `json:"stack_trace,omitempty"`
}

// IsZero returns true if the Extras object is the zero/empty struct value.
func (e DebugExtras) IsZero() bool {
	return e.StackTrace == ""
}

// Link contains a description and hyperlink.
type Link struct {
	URL         string
	Description string
}

// HelpExtras contains helpful hyperlinks for the error.
type HelpExtras struct {
	// Links to help documentation regarding the error.
	Links []Link `json:"links,omitempty"`
}

// IsZero returns true if the Extras object is the zero/empty struct value.
func (e HelpExtras) IsZero() bool {
	return len(e.Links) == 0
}

// RetryExtras contains helpful information for dictating how/why retries can/should happen.
type RetryExtras struct {
	// Delay duration abide by before retrying the failed operation.
	Delay time.Duration
}

// IsZero returns true if the Extras object is the zero/empty struct value.
func (e RetryExtras) IsZero() bool {
	return e.Delay == 0
}

var (
	_ error     = (*errorChain)(nil)
	_ HasAs     = (*errorChain)(nil)
	_ HasIs     = (*errorChain)(nil)
	_ HasUnwrap = (*errorChain)(nil)
)

// errorChain implements the interfaces necessary for errors.Is/As/Unwrap to
// work in a deterministic way. Is/As/Error will work on the error stored
// in the slice at index zero. Upon an Unwrap call, we will return a errorChain
// with a new slice with an index shifted by one.
//
// Based on ideas from https://github.com/hashicorp/go-multierror.
type errorChain []Error

// Error implements the error interface.
func (e errorChain) Error() string {
	if len(e) == 0 {
		return ""
	}
	return e[0].Error()
}

// Unwrap implements errors.Unwrap by returning the next error in the
// errorChain or nil if there are no more errors.
func (e errorChain) Unwrap() error {
	if len(e) <= 1 {
		return nil
	}
	return e[1:]
}

// As implements errors.As by attempting to map to the current value.
func (e errorChain) As(target any) bool {
	if len(e) == 0 {
		return false
	}
	return errors.As(e[0], target)
}

// Is implements errors.Is by comparing the current value directly.
func (e errorChain) Is(target error) bool {
	if len(e) == 0 {
		return false
	}
	return errors.Is(e[0], target)
}

// ErrorGroupFormatter is a function callback that is called by ErrorGroup to
// turn the list of errors into a string.
type ErrorGroupFormatter func([]Error) string

// ErrorGroupFormatterDefault is a basic Formatter that outputs the number of errors
// that occurred along with a bullet point list of the errors.
func ErrorGroupFormatterDefault(errors []Error) string {
	switch len(errors) {
	case 0:
		return ""
	case 1:
		return errors[0].Error()
	default:
		points := make([]string, len(errors))
		for i, err := range errors {
			points[i] = fmt.Sprintf("* %s", err)
		}
		return fmt.Sprintf("\n%s\n\n", strings.Join(points, "\n"))
	}
}

var (
	_ error          = (*ErrorGroup)(nil)
	_ HasUnwrap      = (*ErrorGroup)(nil)
	_ sort.Interface = (*ErrorGroup)(nil)
)

// NewErrorGroup creates a new *ErrorGroup with sane defaults.
func NewErrorGroup(errs ...error) *ErrorGroup {
	eg := &ErrorGroup{
		Errors:    make([]Error, 0, len(errs)),
		Formatter: ErrorGroupFormatterDefault,
	}
	eg.Append(errs...)
	return eg
}

// NewTranslatedErrorGroup creates a new *ErrorGroup with sane defaults
// and translated errors.
func NewTranslatedErrorGroup(translate ErrorTranslate, errs ...error) *ErrorGroup {
	eg := NewErrorGroup(errs...)
	eg.Translate(translate)
	return eg
}

// ErrorGroup stores multiple Error instances.
//
// TODO(ahawker) Flatten JSON output to a single error when group only has one.
type ErrorGroup struct {
	// Errors in the group.
	Errors []Error `json:"errors"`
	// Formatter to convert error group to string representation.
	Formatter ErrorGroupFormatter `json:"-"`
}

// Append adds a new error to the group.
//
// If the given error is not an Error instance, it will be wrapped
// with ErrUndefined.
func (g *ErrorGroup) Append(errs ...error) {
	for _, err := range errs {
		if err == nil {
			continue
		}

		// When given an error that's a group, we want to flatten & merge
		// the items.
		var eg *ErrorGroup
		if errors.As(err, &eg) {
			g.Append(SliceTypeAssert[Error, error](eg.Errors)...)
			continue
		}

		// When given a generic error that isn't Error, wrap it.
		var e Error
		if !errors.As(err, &e) {
			e = ErrUndefined.Wrap(err)
		}

		if e.IsZero() {
			continue
		}

		g.Errors = append(g.Errors, e)
	}
}

// Slice returns a slice of all errors in the group.
func (g *ErrorGroup) Slice() []Error {
	return g.Errors
}

// ErrorOrNil returns an error interface if this Error represents
// a list of errors, or returns nil if the list of errors is empty. This
// function is useful at the end of accumulation to make sure that the value
// returned represents the existence of errors.
func (g *ErrorGroup) ErrorOrNil() error {
	if g == nil {
		return nil
	}
	if g.Errors == nil || len(g.Errors) == 0 {
		return nil
	}
	return g
}

// GroupOrNil returns the ErrorGroup interface if this group represents
// contains one or more errors. If it's empty, nil is returned.
func (g *ErrorGroup) GroupOrNil() *ErrorGroup {
	if g == nil {
		return nil
	}
	if g.Errors == nil || len(g.Errors) == 0 {
		return nil
	}
	return g
}

// Empty will return true if the group is empty.
func (g *ErrorGroup) Empty() bool {
	if g == nil {
		return true
	}
	return len(g.Errors) == 0
}

// Unwrap returns the next error in the group or nil if there are no more errors.
//
// Interface: errors.Unwrap, HasUnwrap.
func (g *ErrorGroup) Unwrap() error {
	// If we have no errors then we do nothing
	if g == nil || len(g.Errors) == 0 {
		return nil
	}

	// If we have exactly one error, we can just return that directly.
	if len(g.Errors) == 1 {
		return g.Errors[0]
	}

	// Shallow copy the errors slice.
	errs := make([]Error, len(g.Errors))
	copy(errs, g.Errors)
	return errorChain(errs)
}

// Error string value of the ErrorGroup struct.
//
// Interface: error.
func (g *ErrorGroup) Error() string {
	return g.Formatter(g.Errors)
}

// Len returns the number of errors in the group.
//
// Interface: sort.Interface.
func (g *ErrorGroup) Len() int {
	return len(g.Errors)
}

// Less determines order for sorting a group.
//
// Interface: sort.Interface.
func (g *ErrorGroup) Less(i, j int) bool {
	return g.Errors[i].Error() < g.Errors[j].Error()
}

// Swap moves errors in the group during sorting.
//
// Interface: sort.Interface.
func (g *ErrorGroup) Swap(i, j int) {
	g.Errors[i], g.Errors[j] = g.Errors[j], g.Errors[i]
}

// Translate performs an in-place translation of errors
// in the group for swapping context.
func (g *ErrorGroup) Translate(translate ErrorTranslate) {
	for i := 0; i < g.Len(); i++ {
		t := translate(g.Errors[i])

		// When given a generic error that isn't Error, wrap it.
		var e Error
		if !errors.As(t, &e) {
			e = ErrUndefined.Wrap(t)
		}

		g.Errors[i] = e
	}
}

// ErrorTranslate defines function that can translate errors between
// two different contexts.
//
// This is commonly used to convert between domain and adapter error types.
type ErrorTranslate func(err error) error

// ErrorJoin is a helper function that will append more errors
// onto an ErrorGroup.
//
// If err is not already an ErrorGroup, then it will be turned into
// one. If any of the errs are ErrorGroup, they will be flattened
// one level into err.
// Any nil errors within errs will be ignored. If err is nil, a new
// *ErrorGroup will be returned containing the given errs.
func ErrorJoin(err error, errs ...error) *ErrorGroup {
	var eg *ErrorGroup

	switch {
	case errors.As(err, &eg):
		eg.Append(errs...)
		return eg
	default:
		eg = NewErrorGroup()
		eg.Append(SliceFlatten([]error{err}, errs)...)
		return eg
	}
}
