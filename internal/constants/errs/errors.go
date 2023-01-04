package errs

import (
	"errors"
)

var ErrStatusInternalServer = errors.New("status internal server error")
var ErrSendMsgGPRC = errors.New("send msg to GPRC error")
var ErrDecrypt = errors.New("decrypt error")
var ErrDecompress = errors.New("decompress error")
var ErrGetJSON = errors.New("get JSON error")
var ErrNotFound = errors.New("not found")
var ErrBadRequest = errors.New("bad request")
var ErrNotImplemented = errors.New("not implemented")
var ErrIPAddressAllowed = errors.New("not IP address allowed")

func StatusError(err error) int32 {
	switch err {
	case ErrStatusInternalServer:
		return 500
	case ErrSendMsgGPRC:
		return 500
	case ErrDecrypt:
		return 500
	case ErrDecompress:
		return 500
	case ErrGetJSON:
		return 500
	case ErrIPAddressAllowed:
		return 500
	case ErrNotFound:
		return 404
	case ErrBadRequest:
		return 400
	case ErrNotImplemented:
		return 501
	default:
		return 200
	}
}
