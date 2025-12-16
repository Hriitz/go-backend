package design

import (
	. "goa.design/goa/v3/dsl"
)

var _ = API("springstreet", func() {
	Title("Spring Street API")
	Description("Backend API for Spring Street - Global investing platform for Indian investors")
	Version("1.0.0")
	Server("api", func() {
		Host("localhost", func() {
			URI("http://localhost:8000")
		})
	})
})

// Common error types
var Unauthorized = Type("Unauthorized", func() {
	Description("Unauthorized access")
	Attribute("message", String, "Error message", func() {
		Example("Unauthorized")
	})
})

var NotFound = Type("NotFound", func() {
	Description("Resource not found")
	Attribute("message", String, "Error message", func() {
		Example("Resource not found")
	})
})

var BadRequest = Type("BadRequest", func() {
	Description("Bad request")
	Attribute("message", String, "Error message", func() {
		Example("Invalid request")
	})
})

// Health check
var _ = Service("health", func() {
	Description("Health check service")
	Method("check", func() {
		Result(HealthResult)
		HTTP(func() {
			GET("/health")
			Response(StatusOK)
		})
	})
})

var HealthResult = ResultType("HealthResult", func() {
	Attribute("status", String, "Service status", func() {
		Example("healthy")
	})
	Attribute("service", String, "Service name", func() {
		Example("Spring Street API")
	})
})

// Authentication service
var _ = Service("auth", func() {
	Description("Authentication service")
	Error("unauthorized", Unauthorized)
	Error("not_found", NotFound)
	Error("bad_request", BadRequest)

	Method("login", func() {
		Description("Authenticate user and return JWT token")
		Payload(LoginPayload)
		Result(LoginResult)
		Error("unauthorized")
		HTTP(func() {
			POST("/api/v1/auth/login")
			Response(StatusOK)
			Response("unauthorized", StatusUnauthorized)
		})
	})

	Method("logout", func() {
		Description("Logout user")
		Security(JWTAuth)
		Payload(LogoutPayload)
		Result(LogoutResult)
		HTTP(func() {
			POST("/api/v1/auth/logout")
			Response(StatusOK)
		})
	})

	Method("me", func() {
		Description("Get current user information")
		Security(JWTAuth)
		Payload(MePayload)
		Result(UserResult)
		Error("unauthorized")
		HTTP(func() {
			GET("/api/v1/auth/me")
			Response(StatusOK)
			Response("unauthorized", StatusUnauthorized)
		})
	})

	Method("create_user", func() {
		Description("Create a new user (Admin only)")
		Security(JWTAuth, func() {
			Scope("admin")
		})
		Payload(CreateUserPayload)
		Result(UserResult)
		Error("bad_request")
		Error("unauthorized")
		HTTP(func() {
			POST("/api/v1/auth/users")
			Response(StatusCreated)
			Response("bad_request", StatusBadRequest)
			Response("unauthorized", StatusUnauthorized)
		})
	})

	Method("list_users", func() {
		Description("List all users (Admin only)")
		Security(JWTAuth, func() {
			Scope("admin")
		})
		Payload(ListUsersPayload)
		Result(ArrayOf(UserResult))
		Error("unauthorized")
		HTTP(func() {
			GET("/api/v1/auth/users")
			Param("skip")
			Param("limit")
			Response(StatusOK)
			Response("unauthorized", StatusUnauthorized)
		})
	})

	Method("get_user", func() {
		Description("Get user by ID (Admin only)")
		Security(JWTAuth, func() {
			Scope("admin")
		})
		Payload(GetUserPayload)
		Result(UserResult)
		Error("not_found")
		Error("unauthorized")
		HTTP(func() {
			GET("/api/v1/auth/users/{id}")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("unauthorized", StatusUnauthorized)
		})
	})

	Method("update_user", func() {
		Description("Update user (Admin only)")
		Security(JWTAuth, func() {
			Scope("admin")
		})
		Payload(UpdateUserPayload)
		Result(UserResult)
		Error("not_found")
		Error("unauthorized")
		HTTP(func() {
			PUT("/api/v1/auth/users/{id}")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("unauthorized", StatusUnauthorized)
		})
	})

	Method("delete_user", func() {
		Description("Delete user (Admin only)")
		Security(JWTAuth, func() {
			Scope("admin")
		})
		Payload(DeleteUserPayload)
		Error("not_found")
		Error("unauthorized")
		HTTP(func() {
			DELETE("/api/v1/auth/users/{id}")
			Response(StatusNoContent)
			Response("not_found", StatusNotFound)
			Response("unauthorized", StatusUnauthorized)
		})
	})
})

// JWT Security
var JWTAuth = JWTSecurity("jwt", func() {
	Description("JWT authentication")
	Scope("admin", "Admin access")
	Scope("staff", "Staff access")
})

// Authentication payloads and results
var LoginPayload = Type("LoginPayload", func() {
	Attribute("username", String, "Username", func() {
		MinLength(1)
		Example("admin")
	})
	Attribute("password", String, "Password", func() {
		MinLength(1)
		Example("password")
	})
	Required("username", "password")
})

var LoginResult = ResultType("LoginResult", func() {
	Attribute("access_token", String, "JWT access token")
	Attribute("token_type", String, "Token type", func() {
		Default("bearer")
		Example("bearer")
	})
	Required("access_token", "token_type")
})

var LogoutPayload = Type("LogoutPayload", func() {
	Token("token", String, "JWT token")
})

var MePayload = Type("MePayload", func() {
	Token("token", String, "JWT token")
})

var LogoutResult = ResultType("LogoutResult", func() {
	Attribute("message", String, "Logout message", func() {
		Example("Successfully logged out")
	})
})

var UserResult = ResultType("UserResult", func() {
	Attribute("id", Int, "User ID")
	Attribute("username", String, "Username")
	Attribute("email", String, "Email address")
	Attribute("full_name", String, "Full name")
	Attribute("is_active", Boolean, "Is user active")
	Attribute("is_admin", Boolean, "Is user admin")
	Attribute("is_staff", Boolean, "Is user staff")
	Attribute("created_at", String, "Creation timestamp")
	Attribute("updated_at", String, "Update timestamp")
	Attribute("last_login", String, "Last login timestamp")
	Required("id", "username", "email", "is_active", "is_admin", "is_staff", "created_at")
})

var CreateUserPayload = Type("CreateUserPayload", func() {
	Token("token", String, "JWT token")
	Attribute("username", String, "Username", func() {
		MinLength(1)
		Example("newuser")
	})
	Attribute("email", String, "Email address", func() {
		Format(FormatEmail)
		Example("user@example.com")
	})
	Attribute("password", String, "Password", func() {
		MinLength(6)
		Example("password123")
	})
	Attribute("full_name", String, "Full name")
	Attribute("is_active", Boolean, "Is user active", func() {
		Default(true)
	})
	Attribute("is_admin", Boolean, "Is user admin", func() {
		Default(false)
	})
	Attribute("is_staff", Boolean, "Is user staff", func() {
		Default(false)
	})
	Required("username", "email", "password")
})

var ListUsersPayload = Type("ListUsersPayload", func() {
	Token("token", String, "JWT token")
	Attribute("skip", Int, "Skip records", func() {
		Default(0)
		Minimum(0)
	})
	Attribute("limit", Int, "Limit records", func() {
		Default(100)
		Minimum(1)
		Maximum(500)
	})
})

var GetUserPayload = Type("GetUserPayload", func() {
	Token("token", String, "JWT token")
	Attribute("id", Int, "User ID")
	Required("id")
})

var UpdateUserPayload = Type("UpdateUserPayload", func() {
	Token("token", String, "JWT token")
	Attribute("id", Int, "User ID")
	Attribute("username", String, "Username")
	Attribute("email", String, "Email address")
	Attribute("full_name", String, "Full name")
	Attribute("is_active", Boolean, "Is user active")
	Attribute("is_admin", Boolean, "Is user admin")
	Attribute("is_staff", Boolean, "Is user staff")
	Attribute("password", String, "Password")
	Required("id")
})

var DeleteUserPayload = Type("DeleteUserPayload", func() {
	Token("token", String, "JWT token")
	Attribute("id", Int, "User ID")
	Required("id")
})

// Investment service
var _ = Service("investment", func() {
	Description("Investment inquiry service")
	Error("not_found", NotFound)
	Error("bad_request", BadRequest)
	Error("unauthorized", Unauthorized)

	Method("create", func() {
		Description("Create a new investment inquiry")
		Payload(InvestmentInquiryCreatePayload)
		Result(InvestmentInquiryResult)
		Error("bad_request")
		HTTP(func() {
			POST("/api/v1/investment/")
			Response(StatusCreated)
			Response("bad_request", StatusBadRequest)
		})
	})

	Method("update_by_phone", func() {
		Description("Update inquiry by phone number")
		Payload(UpdateInquiryByPhonePayload)
		Result(InvestmentInquiryResult)
		Error("not_found")
		HTTP(func() {
			PATCH("/api/v1/investment/by-phone/{phone}")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
		})
	})

	Method("verify", func() {
		Description("Mark inquiry as verified after OTP verification")
		Payload(VerifyInquiryPayload)
		Result(InvestmentInquiryResult)
		Error("not_found")
		HTTP(func() {
			POST("/api/v1/investment/verify/{identifier}")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
		})
	})

	Method("get_by_phone", func() {
		Description("Get inquiry by phone number")
		Payload(GetInquiryByPhonePayload)
		Result(InvestmentInquiryResult)
		Error("not_found")
		HTTP(func() {
			GET("/api/v1/investment/by-phone/{phone}")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
		})
	})

	Method("list", func() {
		Description("List all investment inquiries (Staff/Admin only)")
		Security(JWTAuth, func() {
			Scope("staff")
		})
		Payload(ListInquiriesPayload)
		Result(ArrayOf(InvestmentInquiryResult))
		Error("unauthorized")
		HTTP(func() {
			GET("/api/v1/investment/")
			Param("skip")
			Param("limit")
			Response(StatusOK)
			Response("unauthorized", StatusUnauthorized)
		})
	})

	Method("get", func() {
		Description("Get specific investment inquiry by ID (Staff/Admin only)")
		Security(JWTAuth, func() {
			Scope("staff")
		})
		Payload(GetInquiryPayload)
		Result(InvestmentInquiryResult)
		Error("not_found")
		Error("unauthorized")
		HTTP(func() {
			GET("/api/v1/investment/{id}")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("unauthorized", StatusUnauthorized)
		})
	})
})

var InvestmentInquiryResult = ResultType("InvestmentInquiryResult", func() {
	Attribute("id", Int, "Inquiry ID")
	Attribute("first_name", String, "First name")
	Attribute("last_name", String, "Last name")
	Attribute("phone", String, "Phone number")
	Attribute("email", String, "Email address")
	Attribute("investment_size", String, "Investment size")
	Attribute("current_exposure", String, "Current exposure (comma-separated for multiple selections: direct-stocks, mutual-funds, sip)")
	Attribute("verified", Boolean, "Verification status")
	Attribute("exit_type", String, "Exit type")
	Attribute("created_at", String, "Creation timestamp")
	Attribute("updated_at", String, "Update timestamp")
	Required("id", "verified", "created_at")
})

var InvestmentInquiryCreatePayload = Type("InvestmentInquiryCreatePayload", func() {
	Attribute("phone", String, "Phone number")
	Attribute("first_name", String, "First name")
	Attribute("last_name", String, "Last name")
	Attribute("email", String, "Email address")
	Attribute("investment_size", String, "Investment size")
	Attribute("current_exposure", String, "Current exposure (comma-separated for multiple selections: direct-stocks, mutual-funds, sip)")
	Attribute("exit_type", String, "Exit type", func() {
		Default("abandoned")
		Example("abandoned")
	})
})

var UpdateInquiryByPhonePayload = Type("UpdateInquiryByPhonePayload", func() {
	Attribute("phone", String, "Phone number")
	Attribute("first_name", String, "First name")
	Attribute("last_name", String, "Last name")
	Attribute("email", String, "Email address")
	Attribute("investment_size", String, "Investment size")
	Attribute("current_exposure", String, "Current exposure (comma-separated for multiple selections: direct-stocks, mutual-funds, sip)")
	Required("phone")
})

var VerifyInquiryPayload = Type("VerifyInquiryPayload", func() {
	Attribute("identifier", String, "Phone number or email")
	Required("identifier")
})

var GetInquiryByPhonePayload = Type("GetInquiryByPhonePayload", func() {
	Attribute("phone", String, "Phone number")
	Required("phone")
})

var ListInquiriesPayload = Type("ListInquiriesPayload", func() {
	Token("token", String, "JWT token")
	Attribute("skip", Int, "Skip records", func() {
		Default(0)
		Minimum(0)
	})
	Attribute("limit", Int, "Limit records", func() {
		Default(100)
		Minimum(1)
		Maximum(500)
	})
})

var GetInquiryPayload = Type("GetInquiryPayload", func() {
	Token("token", String, "JWT token")
	Attribute("id", Int, "Inquiry ID")
	Required("id")
})

// OTP service
var _ = Service("otp", func() {
	Description("OTP (One-Time Password) service")
	Error("bad_request", BadRequest)

	Method("send", func() {
		Description("Send OTP to phone number or email")
		Payload(SendOTPPayload)
		Result(SendOTPResult)
		Error("bad_request")
		HTTP(func() {
			POST("/api/v1/otp/send")
			Response(StatusOK)
			Response("bad_request", StatusBadRequest)
		})
	})

	Method("verify", func() {
		Description("Verify OTP code")
		Payload(VerifyOTPPayload)
		Result(VerifyOTPResult)
		Error("bad_request")
		HTTP(func() {
			POST("/api/v1/otp/verify")
			Response(StatusOK)
			Response("bad_request", StatusBadRequest)
		})
	})

	Method("check", func() {
		Description("Check verification status")
		Payload(CheckVerificationPayload)
		Result(CheckVerificationResult)
		HTTP(func() {
			POST("/api/v1/otp/check")
			Response(StatusOK)
		})
	})
})

var SendOTPPayload = Type("SendOTPPayload", func() {
	Attribute("phone_number", String, "Phone number")
	Attribute("email", String, "Email address")
})

var SendOTPResult = ResultType("SendOTPResult", func() {
	Attribute("message", String, "Response message")
	Attribute("phone_number", String, "Phone number")
	Attribute("expires_in_minutes", Int, "OTP expiration in minutes", func() {
		Default(10)
	})
	Required("message", "phone_number", "expires_in_minutes")
})

var VerifyOTPPayload = Type("VerifyOTPPayload", func() {
	Attribute("phone_number", String, "Phone number")
	Attribute("email", String, "Email address")
	Attribute("otp_code", String, "6-digit OTP code", func() {
		MinLength(6)
		MaxLength(6)
		Example("123456")
	})
	Required("otp_code")
})

var VerifyOTPResult = ResultType("VerifyOTPResult", func() {
	Attribute("message", String, "Response message")
	Attribute("phone_number", String, "Phone number")
	Attribute("verified", Boolean, "Verification status", func() {
		Default(true)
	})
	Required("message", "phone_number", "verified")
})

var CheckVerificationPayload = Type("CheckVerificationPayload", func() {
	Attribute("phone_number", String, "Phone number", func() {
		MinLength(10)
		MaxLength(20)
	})
	Required("phone_number")
})

var CheckVerificationResult = ResultType("CheckVerificationResult", func() {
	Attribute("phone_number", String, "Phone number")
	Attribute("verified", Boolean, "Verification status")
	Required("phone_number", "verified")
})

// Contact service
var _ = Service("contact", func() {
	Description("Contact form service")
	Error("bad_request", BadRequest)
	Error("unauthorized", Unauthorized)

	Method("submit", func() {
		Description("Submit contact form")
		Payload(ContactSubmitPayload)
		Result(ContactSubmitResult)
		Error("bad_request")
		HTTP(func() {
			POST("/api/v1/contact/submit")
			Response(StatusOK)
			Response("bad_request", StatusBadRequest)
		})
	})

	Method("list", func() {
		Description("List all contact inquiries (Staff/Admin only)")
		Security(JWTAuth, func() {
			Scope("staff")
		})
		Payload(ListContactInquiriesPayload)
		Result(ArrayOf(ContactInquiryResult))
		Error("unauthorized")
		HTTP(func() {
			GET("/api/v1/contact/")
			Param("skip")
			Param("limit")
			Response(StatusOK)
			Response("unauthorized", StatusUnauthorized)
		})
	})
})

var ContactSubmitPayload = Type("ContactSubmitPayload", func() {
	Attribute("name", String, "Full name", func() {
		MinLength(2)
		MaxLength(100)
		Example("John Doe")
	})
	Attribute("email", String, "Email address", func() {
		Format(FormatEmail)
		Example("john@example.com")
	})
	Attribute("phone", String, "Phone number (optional)")
	Attribute("message", String, "Message", func() {
		MinLength(1)
		MaxLength(5000)
		Example("I'm interested in learning more about global investing.")
	})
	Required("name", "email", "message")
})

var ContactSubmitResult = ResultType("ContactSubmitResult", func() {
	Attribute("id", Int, "Contact inquiry ID")
	Attribute("message", String, "Success message")
	Required("id", "message")
})

var ListContactInquiriesPayload = Type("ListContactInquiriesPayload", func() {
	Token("token", String, "JWT token")
	Attribute("skip", Int, "Skip records", func() {
		Default(0)
		Minimum(0)
	})
	Attribute("limit", Int, "Limit records", func() {
		Default(100)
		Minimum(1)
		Maximum(500)
	})
})

var ContactInquiryResult = ResultType("ContactInquiryResult", func() {
	Attribute("id", Int, "Contact inquiry ID")
	Attribute("name", String, "Full name")
	Attribute("email", String, "Email address")
	Attribute("phone", String, "Phone number")
	Attribute("message", String, "Message content")
	Attribute("status", String, "Status (new, read, replied)")
	Attribute("created_at", String, "Creation timestamp")
	Attribute("updated_at", String, "Update timestamp")
	Required("id", "name", "email", "message", "status", "created_at")
})
