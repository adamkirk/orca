package extension

import "errors"

var (
	ErrUnknownService       = errors.New("unknown service")
	ErrNoServices           = errors.New("no services to run the extension in")
	ErrProjectNotCloned     = errors.New("project isn't cloned")
	ErrUnexpectedArgs       = errors.New("extension doesn't take arguments")
	ErrSourceNotFound       = errors.New("source not found")
	ErrTargetExists         = errors.New("target already exists, and isn't the expected symlink")
	ErrSymlinksNotSupported = errors.New("the filesystem doesn't support symlinks")
	ErrTrustDeclined        = errors.New("the extensions weren't trusted")
	ErrTrustNeeded          = errors.New("the extensions need trusting")
)
