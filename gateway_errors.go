package volcenginex

import "strconv"

// String codes documented by the Volcengine machine translation gateway.
const (
	// CodeUndefinedError is gateway error 100001.
	CodeUndefinedError = "UndefinedError"
	// CodeMissingParameter is gateway error 100002.
	CodeMissingParameter = "MissingParameter"
	// CodeMissingAuthenticationToken is gateway error 100003.
	CodeMissingAuthenticationToken = "MissingAuthenticationToken"
	// CodeMissingRequestInfo is gateway error 100004.
	CodeMissingRequestInfo = "MissingRequestInfo"
	// CodeMissingSignature is gateway error 100005.
	CodeMissingSignature = "MissingSignature"
	// CodeInvalidTimestamp is gateway error 100006.
	CodeInvalidTimestamp = "InvalidTimestamp"
	// CodeServiceUnavailable is gateway error 100007.
	CodeServiceUnavailable = "ServiceUnavailable"
	// CodeInvalidActionOrVersion is gateway error 100008.
	CodeInvalidActionOrVersion = "InvalidActionOrVersion"
	// CodeInvalidClientTokenId is gateway error 100009.
	CodeInvalidClientTokenId = "InvalidClientTokenId"
	// CodeSignatureDoesNotMatch is gateway error 100010.
	CodeSignatureDoesNotMatch = "SignatureDoesNotMatch"
	// CodeServiceShouldCallSelfApi is gateway error 100011.
	CodeServiceShouldCallSelfApi = "ServiceShouldCallSelfApi"
	// CodeLackPolicy is gateway error 100012.
	CodeLackPolicy = "LackPolicy"
	// CodeAccessDenied is gateway error 100013.
	CodeAccessDenied = "AccessDenied"
	// CodeInternalError is gateway error 100014.
	CodeInternalError = "InternalError"
	// CodeFailToConnect is gateway error 100015.
	CodeFailToConnect = "FailToConnect"
	// CodeInternalServiceTimeout is gateway error 100016.
	CodeInternalServiceTimeout = "InternalServiceTimeout"
	// CodeInnerApiNeedAccountInfo is gateway error 100017.
	CodeInnerApiNeedAccountInfo = "InnerApiNeedAccountInfo"
	// CodeFlowLimitExceeded is gateway error 100018.
	CodeFlowLimitExceeded = "FlowLimitExceeded"
	// CodeServiceUnavailableTemp is gateway error 100019.
	CodeServiceUnavailableTemp = "ServiceUnavailableTemp"
	// CodeMethodNotAllowed is gateway error 100020.
	CodeMethodNotAllowed = "MethodNotAllowed"
	// CodeLackProjectPolicy is gateway error 100021.
	CodeLackProjectPolicy = "LackProjectPolicy"
)

// GatewayErrorInfo describes an error listed in the machine translation gateway documentation.
type GatewayErrorInfo struct {
	_ struct{} // Require keyed literals outside this package.

	Code            string
	CodeN           int
	MessageTemplate string
}

// LookupGatewayError finds the documented error by its string code or decimal CodeN.
// The message template describes the documented error. Error.Message carries the response message.
// An unrecognized code returns a zero value and false.
func LookupGatewayError(code string) (GatewayErrorInfo, bool) {
	if info, ok := gatewayErrors[code]; ok {
		return info, true
	}
	number, err := strconv.Atoi(code)
	if err == nil {
		for _, info := range gatewayErrors {
			if info.CodeN == number {
				return info, true
			}
		}
	}
	return GatewayErrorInfo{}, false
}

var gatewayErrors = map[string]GatewayErrorInfo{
	CodeUndefinedError:             {Code: CodeUndefinedError, CodeN: 100001, MessageTemplate: "Undefined Internal Error. Pls Contact With Admin."},
	CodeMissingParameter:           {Code: CodeMissingParameter, CodeN: 100002, MessageTemplate: "The request is missing %s parameter."},
	CodeMissingAuthenticationToken: {Code: CodeMissingAuthenticationToken, CodeN: 100003, MessageTemplate: "Request is missing Authentication Token"},
	CodeMissingRequestInfo:         {Code: CodeMissingRequestInfo, CodeN: 100004, MessageTemplate: "The request is missing %s information."},
	CodeMissingSignature:           {Code: CodeMissingSignature, CodeN: 100005, MessageTemplate: "The request is missing signature."},
	CodeInvalidTimestamp:           {Code: CodeInvalidTimestamp, CodeN: 100006, MessageTemplate: "The Signature of the request is expired"},
	CodeServiceUnavailable:         {Code: CodeServiceUnavailable, CodeN: 100007, MessageTemplate: "Ths service[%s] is unavailable."},
	CodeInvalidActionOrVersion:     {Code: CodeInvalidActionOrVersion, CodeN: 100008, MessageTemplate: "Could not find operation %s for version %s"},
	CodeInvalidClientTokenId:       {Code: CodeInvalidClientTokenId, CodeN: 100009, MessageTemplate: "The security token[%s] included in the request is invalid."},
	CodeSignatureDoesNotMatch:      {Code: CodeSignatureDoesNotMatch, CodeN: 100010, MessageTemplate: "The request signature we calculated does not match the signature you provided. Check your Secret Access Key and signing method. Consult the service documentation for details."},
	CodeServiceShouldCallSelfApi:   {Code: CodeServiceShouldCallSelfApi, CodeN: 100011, MessageTemplate: "The service(%s) should call self-api"},
	CodeLackPolicy:                 {Code: CodeLackPolicy, CodeN: 100012, MessageTemplate: "Request was rejected because of lack of policy."},
	CodeAccessDenied:               {Code: CodeAccessDenied, CodeN: 100013, MessageTemplate: "User is not authorized to perform: %s on resource: %s"},
	CodeInternalError:              {Code: CodeInternalError, CodeN: 100014, MessageTemplate: "Service has some internal Error. Pls Contact With Admin"},
	CodeFailToConnect:              {Code: CodeFailToConnect, CodeN: 100015, MessageTemplate: "Top Fail to Connect to Internal Service"},
	CodeInternalServiceTimeout:     {Code: CodeInternalServiceTimeout, CodeN: 100016, MessageTemplate: "Internal Service is timeout. Pls Contact With Admin"},
	CodeInnerApiNeedAccountInfo:    {Code: CodeInnerApiNeedAccountInfo, CodeN: 100017, MessageTemplate: "The inner request should have account info"},
	CodeFlowLimitExceeded:          {Code: CodeFlowLimitExceeded, CodeN: 100018, MessageTemplate: "Request was rejected because the request speed of this openAPI is beyond the current flow control limit."},
	CodeServiceUnavailableTemp:     {Code: CodeServiceUnavailableTemp, CodeN: 100019, MessageTemplate: "Service is busy, please try again later."},
	CodeMethodNotAllowed:           {Code: CodeMethodNotAllowed, CodeN: 100020, MessageTemplate: "HTTP Method Not Allowed"},
	CodeLackProjectPolicy:          {Code: CodeLackProjectPolicy, CodeN: 100021, MessageTemplate: "Request was rejected because of lack of project policy."},
}
