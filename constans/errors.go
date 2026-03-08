package constans

import "github.com/ariandi/gocom"

var ErrEmailEmpty *gocom.CodedError = gocom.NewError(2001, "Email cannot empty")
var ErrRequestEmpty *gocom.CodedError = gocom.NewError(2003, "Request cannot empty")
var ErrFullNameEmpty *gocom.CodedError = gocom.NewError(2004, "name cannot empty")
var ErrClientEmpty *gocom.CodedError = gocom.NewError(2005, "Client id cannot empty")
var ErrPasswordEmpty *gocom.CodedError = gocom.NewError(2006, "password cannot empty")
var ErrInvalidOldPassword *gocom.CodedError = gocom.NewError(2007, "Invalid old password")
var ErrInvalidUser *gocom.CodedError = gocom.NewError(2008, "Invalid user")
var ErrUnableToParse *gocom.CodedError = gocom.NewError(2009, "Unable to parse")
var ErrInvalidRequest *gocom.CodedError = gocom.NewError(2010, "Invalid request")
var ErrParentEmpty *gocom.CodedError = gocom.NewError(2011, "parent id cannot empty")

var ErrCreateExcelHeader *gocom.CodedError = gocom.NewError(3001, "error create excel header")
var ErrCreateExcelData *gocom.CodedError = gocom.NewError(3002, "error create excel data")
