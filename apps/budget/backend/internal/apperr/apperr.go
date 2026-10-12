package apperr

type Kind int

const (
	Invalid Kind = iota
	Unauthorized
	Forbidden
	NotFound
	Conflict
	Internal
)

type Error struct {
	Kind Kind
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func NewInvalid(msg string) *Error      { return &Error{Kind: Invalid, Msg: msg} }
func NewUnauthorized(msg string) *Error { return &Error{Kind: Unauthorized, Msg: msg} }
func NewForbidden(msg string) *Error    { return &Error{Kind: Forbidden, Msg: msg} }
func NewNotFound(msg string) *Error     { return &Error{Kind: NotFound, Msg: msg} }
func NewConflict(msg string) *Error     { return &Error{Kind: Conflict, Msg: msg} }
func NewInternal(msg string) *Error     { return &Error{Kind: Internal, Msg: msg} }
