package handler

import (
	"errors"
	"net/http"
)

var (
	AccessDeniedException       error = errors.New("AccessDeniedException")
	InternalFailure             error = errors.New("InternalFailure")
	InvalidAction               error = errors.New("InvalidAction")
	InvalidClientTokenId        error = errors.New("InvalidClientTokenId")
	InvalidParameterCombination error = errors.New("InvalidParameterCombination")
	MissingAction               error = errors.New("MissingAction")
	MissingAuthenticationToken  error = errors.New("MissingAuthenticationToken")
	MissingParameter            error = errors.New("MissingParameter")
	NotAuthorized               error = errors.New("NotAuthorized")
	MalformedRequestBody        error = errors.New("MalformedRequestBody")
	InvalidParameterValue       error = errors.New("InvalidParameterValue")
)

var RetCodes = map[error]int{
	AccessDeniedException:       http.StatusForbidden,
	InternalFailure:             http.StatusInternalServerError,
	MissingAction:               http.StatusBadRequest,
	InvalidAction:               http.StatusBadRequest,
	InvalidClientTokenId:        http.StatusBadRequest,
	InvalidParameterCombination: http.StatusBadRequest,

	MissingAuthenticationToken: http.StatusUnauthorized,
	NotAuthorized:              http.StatusUnauthorized,
	MalformedRequestBody:       http.StatusBadRequest,
}


