// 分类原则：责任归属
// 1. 请求参数不恰当、操作/特性不支持，划到client类，以c_开头，寄希望于用户自查、或SPT协助解决
// 2. 调用三方API接口失败（RetCode不为0、结果不合预期），划到third类，以t_开头，可请兄弟部门同学帮忙debug
// 3. 除此之外，划到server类，以s_开头，主机同学通过log自查
// 4. 存在某些无法严格界限的错误，如资源配额类，可能是主机资源配额不足，也可能三方资源配额不足，具体问题具体分析

// 命名风格：统一（可抽象为2类模板
// 1. [Action]Error/Failed/Timeout等，约定俗成，Error用于server类，Failed用于third类
// 2. [Object]Not[Expected]、[Object][Negative]等

// 建议：
// 1. 尽量少用缩写，如NotSpt，没有这种用法；如IG，无法看出含义
// 2. 尽量为错误码添加中文注释

package errors

type ErrorCode int

var (
	// Common
	_codeServiceUnavailable  ErrorCode = 150 // 服务不可用
	_codeMissingAction       ErrorCode = 160 // 参数缺少action
	_codeParamsRangeError    ErrorCode = 220 // 参数不在范围内
	_codeMissingParams       ErrorCode = 210 // 参数缺少
	_codeParamsError         ErrorCode = 230 // 参数无效
	_codeParamsConflictError ErrorCode = 230 // 参数冲突
	_codeParamsFormatError   ErrorCode = 231 // 参数格式不对
	_codePermissionError     ErrorCode = 240 // 权限出错
	_codeLackOfBalance       ErrorCode = 520 // 余额不足

	// Client
	// 1. Param
	c_SignatureInvalid   ErrorCode = 8046 // 签名无效
	c_AddrHashNoNonce    ErrorCode = 8047 // 没有Nonce
	c_NonceInvalid       ErrorCode = 8048 // Nonce无效
	c_LoginModeInvalid   ErrorCode = 8061 // 登陆模式无效
	c_MissingLoginCookie ErrorCode = 8062 // 没有携带登录的cookie
	c_LoginCookieInvalid ErrorCode = 8063 // 登录的cookie无效
	c_PasswordInvalid    ErrorCode = 8314 // 密码无效
	c_StorageTypeInvalid ErrorCode = 8371 // 存储类型无效

	// Server
	s_ActionError                ErrorCode = 8433 // [行为]出错（目前被滥用，考虑将一部分拆到s_CallThirdApiError
	s_ActionTimeout              ErrorCode = 8438 // [行为]超时
	s_GetZonesFailed             ErrorCode = 8439 // [行为]从数据库访问zone信息失败
	s_RedisKeyConflict           ErrorCode = 8440 // [行为]redis里的Key冲突
	s_ListDeployments            ErrorCode = 8441 // [行为]从数据库访问deployments信息失败
	s_SaveSessionFailed          ErrorCode = 8442 // [行为]保存session失败
	s_GetCloudsFailed            ErrorCode = 8450 // [行为]从数据库访问cloud信息失败
	s_GetChainCombinationsFailed ErrorCode = 8451 // [行为]从数据库访问ChainCombination信息失败

	// Third Party
	t_GetBuyPriceFailed ErrorCode = 8090 // 获取购买价格失败

	// contract
	t_GetRealTimeRewardsFailed ErrorCode = 9000 // 从合约获取奖励失败

	// db
	d_OperateDbFailed ErrorCode = 10000 // 操作db失败
)

// Common begin
func ServiceUnavailable(a ...interface{}) Error {
	return Error{code: _codeServiceUnavailable, items: a, message: "Service unavailable"}
}

func MissAction(a ...interface{}) Error {
	return Error{code: _codeMissingAction, items: a, message: "Missing Action"}
}

func ParamsRangeError(a ...interface{}) Error {
	return Error{code: _codeParamsRangeError, items: a, message: "Params [%s], should be in [%+v, %+v]"}
}

func MissingParams(a ...interface{}) Error {
	return Error{code: _codeMissingParams, items: a, message: "Missing params [%s]"}
}

func ParamsError(a ...interface{}) Error {
	return Error{code: _codeParamsError, items: a, message: "Params [%s] not available"}
}

func ParamsJudgeError(a ...interface{}) Error {
	return Error{code: _codeParamsError, items: a, message: "Params not available"}
}

func ParamsFormatError(a ...interface{}) Error {
	return Error{code: _codeParamsFormatError, items: a, message: "Params [%s] format error"}
}

func ParamsConflictError(a ...interface{}) Error {
	return Error{code: _codeParamsConflictError, items: a, message: "Param [%s] conflict with param [%s]"}
}

func PermissionError(a ...interface{}) Error {
	return Error{code: _codePermissionError, items: a, message: "Permission error"}
}

func LackOfBalance(a ...interface{}) Error {
	return Error{code: _codeLackOfBalance, items: a, message: "Lack of balance"}
}

func SignatureInvalid(a ...interface{}) Error {
	return Error{code: c_SignatureInvalid, items: a, message: "Signature invalid"}
}

func AddrHasNoNonce(a ...interface{}) Error {
	return Error{code: c_AddrHashNoNonce, items: a, message: "Addr [%s] has no nonce"}
}

func NonceInvalid(a ...interface{}) Error {
	return Error{code: c_NonceInvalid, items: a, message: "Nonce [%s] invalid or expired"}
}

func MissingLoginCookie(a ...interface{}) Error {
	return Error{code: c_MissingLoginCookie, items: a, message: "Missing login cookie"}
}

func LoginCookieInvalid(a ...interface{}) Error {
	return Error{code: c_LoginCookieInvalid, items: a, message: "Login cookie [%s] invalid or expired"}
}

func ActionError(a ...interface{}) Error {
	return Error{code: s_ActionError, items: a, message: "Action error: "}
}

func ActionTimeout(a ...interface{}) Error {
	return Error{code: s_ActionTimeout, items: a, message: "Action timeout: "}
}

func GetBuyPriceFailed(a ...interface{}) Error {
	return Error{code: t_GetBuyPriceFailed, items: a, message: "Get an error while get price"}
}

func GetZonesFailed(a ...interface{}) Error {
	return Error{code: s_GetZonesFailed, items: a, message: "get zones failed"}
}

func GetCloudsFailed(a ...interface{}) Error {
	return Error{code: s_GetCloudsFailed, items: a, message: "get clouds failed"}
}

func GetChainCombinationsFailed(a ...interface{}) Error {
	return Error{code: s_GetChainCombinationsFailed, items: a, message: "get ChainCombinations failed"}
}

func RedisKeyConflict(a ...interface{}) Error {
	return Error{code: s_RedisKeyConflict, items: a, message: "redis key conflict"}
}

func SaveSessionFailed(a ...interface{}) Error {
	return Error{code: s_SaveSessionFailed, items: a, message: "save session failed"}
}

func ListDeploymentsFailed(a ...interface{}) Error {
	return Error{code: s_ListDeployments, items: a, message: "list deployments failed"}
}

func GetRealTimeRewardsFailed(a ...interface{}) Error {
	return Error{code: t_GetRealTimeRewardsFailed, items: a, message: "GetRealTimeRewards From Contract Failed"}
}

func OperateDbFailed(a ...interface{}) Error {
	return Error{code: d_OperateDbFailed, items: a, message: "OperateDb Failed"}
}
