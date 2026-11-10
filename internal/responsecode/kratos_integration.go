package responsecode

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/errors"
)

// KratosErrorConverter Kratos错误转换器
// 将响应码转换为Kratos错误

// NewKratosError 创建带响应码的Kratos错误
// 错误消息从 codes.go 的映射表中自动获取
func NewKratosError(code int) error {
	reason := getKratosReason(code)
	message := getCodeMessage(code)

	// 根据错误码确定HTTP状态码
	var httpCode int
	switch {
	case code == 40002 || code == 40003 || code == 40004:
		httpCode = 401
	case code == 40005 || code == 40006 || code == 40007 || code == 40008:
		httpCode = 403
	case code == 500 || code == 10001 || code == 10002 || code == 10003 || code == 10004 || code == 80001 || code == 90001 || code == 90002 || code == 90003 || code == 90004 || code == 90005 || code == 90006 || code == 90007 || code == 90008 || code == 90009 || code == 90010 || code == 90011 || code == 90012 || code == 90013 || code == 90014 || code == 90015 || code == 90016 || code == 90017 || code == 90018:
		httpCode = 500
	default:
		httpCode = 400
	}

	// 创建错误，并将自定义错误码放在metadata中
	return errors.New(httpCode, reason, message).WithMetadata(map[string]string{
		"custom_code": fmt.Sprintf("%d", code),
	})
}

// getKratosReason 根据响应码获取Kratos错误原因
func getKratosReason(code int) string {
	switch code {
	case 40002:
		return "MISSING_AUTH_TOKEN"
	case 40003:
		return "INVALID_AUTH_TOKEN"
	case 40004:
		return "AUTH_TOKEN_EXPIRED"
	case 20003:
		return "INVALID_CREDENTIALS"
	case 20004:
		return "ACCOUNT_DISABLED"
	case 40008:
		return "PERMISSION_DENIED"
	case 40005:
		return "RESOURCE_ACCESS_DENIED"
	case 40006:
		return "INVALID_CIPHERTEXT"
	case 40007:
		return "SECRET_IS_EMPTY"
	case 400:
		return "INVALID_PARAMETER"
	case 401:
		return "TOO_MANY_REQUESTS"
	case 20002:
		return "USER_NOT_FOUND"
	case 61001:
		return "ORDER_NOT_FOUND"
	case 60002:
		return "SUBSCRIBE_NOT_FOUND"
	case 61002:
		return "PAYMENT_NOT_FOUND"
	case 30002:
		return "SERVER_NOT_FOUND"
	case 50001:
		return "COUPON_NOT_FOUND"
	case 90017:
		return "DEVICE_NOT_FOUND"
	case 30004:
		return "SERVER_GROUP_NOT_FOUND"
	case 20001:
		return "USER_ALREADY_EXISTS"
	case 90011:
		return "DUPLICATE_EMAIL"
	case 90012:
		return "TELEPHONE_EXIST"
	case 60003:
		return "SUBSCRIBE_ALREADY_EXISTS"
	case 30001:
		return "SERVER_ALREADY_EXISTS"
	case 61003:
		return "ORDER_CANNOT_CANCEL"
	case 50005:
		return "COUPON_EXPIRED"
	case 50003:
		return "COUPON_NOT_AVAILABLE"
	case 50002:
		return "COUPON_USED_UP"
	case 50004:
		return "COUPON_USER_LIMIT_EXCEEDED"
	case 20005:
		return "INSUFFICIENT_BALANCE"
	case 20010:
		return "USER_COMMISSION_NOT_ENOUGH"
	case 60001:
		return "SUBSCRIBE_EXPIRED"
	case 61005:
		return "TRAFFIC_EXCEEDED"
	case 60004:
		return "SUBSCRIBE_IN_USE"
	case 60005:
		return "SINGLE_SUBSCRIBE_MODE_EXCEEDS_LIMIT"
	case 60006:
		return "SUBSCRIBE_QUOTA_LIMIT"
	case 60007:
		return "SUBSCRIBE_OUT_OF_STOCK"
	case 61004:
		return "INSUFFICIENT_OF_PERIOD"
	case 70001:
		return "VERIFY_CODE_ERROR"
	case 80001:
		return "QUEUE_ENQUEUE_ERROR"
	case 90001:
		return "DEBUG_MODE_ERROR"
	case 90002:
		return "SMS_SEND_FAILED"
	case 90003:
		return "SMS_NOT_ENABLED"
	case 90004:
		return "EMAIL_NOT_ENABLED"
	case 90005:
		return "GET_AUTHENTICATOR_ERROR"
	case 90006:
		return "AUTHENTICATOR_NOT_SUPPORTED"
	case 90007:
		return "TELEPHONE_AREA_CODE_EMPTY"
	case 90008:
		return "PASSWORD_EMPTY"
	case 90009:
		return "AREA_CODE_EMPTY"
	case 90010:
		return "PASSWORD_OR_VERIFICATION_CODE_REQUIRED"
	case 90013:
		return "DEVICE_EXIST"
	case 90014:
		return "TELEPHONE_ERROR"
	case 90015:
		return "TODAY_SEND_COUNT_EXCEEDS_LIMIT"
	case 90016:
		return "INVALID_EMAIL"
	case 90018:
		return "USERID_NOT_MATCH"
	case 20006:
		return "STOP_REGISTER"
	case 20007:
		return "TELEGRAM_NOT_BOUND"
	case 20008:
		return "USER_NOT_BIND_OAUTH"
	case 20009:
		return "INVITE_CODE_ERROR"
	case 20011:
		return "REGISTER_IP_LIMIT"
	case 30003:
		return "NODE_GROUP_EXIST"
	case 30005:
		return "NODE_GROUP_NOT_EMPTY"
	case 30006:
		return "PROTOCOL_PORT_OCCUPIED"
	case 30007:
		return "PROTOCOL_INSTANCE_SHARED"
	case 10001:
		return "DATABASE_QUERY_FAILED"
	case 10002:
		return "DATABASE_UPDATE_FAILED"
	case 10003:
		return "DATABASE_INSERT_FAILED"
	case 10004:
		return "DATABASE_DELETE_FAILED"
	case 500:
		return "INTERNAL_ERROR"
	}
	return "UNKNOWN_ERROR"
}

// ==== 便捷的Kratos错误创建方法 ====

// CreateKratosErrorFromCode 根据响应码创建Kratos错误
func CreateKratosErrorFromCode(code int) error {
	return NewKratosError(code)
}

// ==== 常用错误快捷方法 ====

// ErrUnauthorized 未认证错误
func ErrUnauthorized() error {
	return NewKratosError(ErrUserNotAuthenticated)
}

// ErrForbidden 无权限错误
func ErrForbidden() error {
	return NewKratosError(ErrPermissionDenied)
}

// ==== Order相关错误 ====

// ErrOrderIDRequired 订单ID必需
func ErrOrderIDRequired() error {
	return NewKratosError(ErrInvalidOrderID)
}

// ErrOrderCreateFailed 订单创建失败
func ErrOrderCreateFailed() error {
	return NewKratosError(ErrDatabaseInsert)
}

// ErrOrderUpdateFailed 订单更新失败
func ErrOrderUpdateFailed() error {
	return NewKratosError(ErrDatabaseUpdate)
}

// ErrOrderListFailed 订单列表获取失败
func ErrOrderListFailed() error {
	return NewKratosError(ErrDatabaseQuery)
}

// ==== 通用参数验证错误 ====

// ErrUserIDRequired 用户ID必需
func ErrUserIDRequired() error {
	return NewKratosError(ErrInvalidUserID)
}

// ErrInvalidParam 无效参数
func ErrInvalidParam() error {
	return NewKratosError(ErrInvalidParameter)
}

// ==== Task相关错误 ====

// ErrTaskCreateFailed 任务创建失败
func ErrTaskCreateFailed() error {
	return NewKratosError(ErrDatabaseInsert)
}

// ErrTaskUpdateFailed 任务更新失败
func ErrTaskUpdateFailed() error {
	return NewKratosError(ErrDatabaseUpdate)
}

// ErrTaskListFailed 任务列表获取失败
func ErrTaskListFailed() error {
	return NewKratosError(ErrDatabaseQuery)
}

// ErrTaskQueryFailed 任务查询失败
func ErrTaskQueryFailed() error {
	return NewKratosError(ErrDatabaseQuery)
}
