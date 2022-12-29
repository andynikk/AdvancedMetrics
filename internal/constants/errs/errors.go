package errs

import "errors"

var ErrStatusInternalServer = errors.New("status internal server error")
var ErrSendMsgGPRC = errors.New("send msg to GPRC error")
var ErrDecrypt = errors.New("decrypt error")
var ErrDecompress = errors.New("decompress error")
