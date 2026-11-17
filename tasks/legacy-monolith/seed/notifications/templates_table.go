package notifications

// Template ids are part of the contract with the email vendor; do not
// rename them without migrating the vendor side first.
const (
	TemplateOrderPlaced           = "order_placed"
	TemplateOrderShipped          = "order_shipped"
	TemplateOrderDelivered        = "order_delivered"
	TemplateOrderCancelled        = "order_cancelled"
	TemplatePaymentFailed         = "payment_failed"
	TemplatePaymentCaptured       = "payment_captured"
	TemplateRefundIssued          = "refund_issued"
	TemplateRefundRejected        = "refund_rejected"
	TemplateReturnAuthorized      = "return_authorized"
	TemplateReturnReceived        = "return_received"
	TemplateLoyaltyPointsEarned   = "loyalty_points_earned"
	TemplateLoyaltyPointsRedeemed = "loyalty_points_redeemed"
	TemplateLoyaltyTierChanged    = "loyalty_tier_changed"
	TemplateCouponExpiring        = "coupon_expiring"
	TemplateCartAbandoned         = "cart_abandoned"
	TemplatePasswordReset         = "password_reset"
	TemplateAccountSuspended      = "account_suspended"
	TemplateBackInStock           = "back_in_stock"
	TemplatePriceDrop             = "price_drop"
	TemplateReviewRequest         = "review_request"
	TemplateSupplierPoApproved    = "supplier_po_approved"
	TemplateInvoiceReady          = "invoice_ready"
	TemplateInvoiceOverdue        = "invoice_overdue"
	TemplateDeliverySlotBooked    = "delivery_slot_booked"
	TemplateDeliverySlotChanged   = "delivery_slot_changed"
)

var templates = []tmpl{
	{ID: TemplateOrderPlaced, Channel: "email", Subject: "Order placed", Body: "Dear {{name}},\n\nYour order placed update for {{ref}} is ready.\n\nThanks for shopping with us"},
	{ID: TemplateOrderShipped, Channel: "email", Subject: "Order shipped", Body: "Hello {{name}},\n\nYour order shipped update for {{ref}} is ready.\n\nThe Shop team"},
	{ID: TemplateOrderDelivered, Channel: "email", Subject: "Order delivered", Body: "Hi {{name}},\n\nYour order delivered update for {{ref}} is ready.\n\nThanks for shopping with us"},
	{ID: TemplateOrderCancelled, Channel: "email", Subject: "Order cancelled", Body: "Hi {{name}},\n\nYour order cancelled update for {{ref}} is ready.\n\nThanks for shopping with us"},
	{ID: TemplatePaymentFailed, Channel: "push", Subject: "Payment failed", Body: "Dear {{name}},\n\nYour payment failed update for {{ref}} is ready.\n\nThe Shop team"},
	{ID: TemplatePaymentCaptured, Channel: "email", Subject: "Payment captured", Body: "Dear {{name}},\n\nYour payment captured update for {{ref}} is ready.\n\nThe Shop team"},
	{ID: TemplateRefundIssued, Channel: "email", Subject: "Refund issued", Body: "Hi {{name}},\n\nYour refund issued update for {{ref}} is ready.\n\nCustomer care"},
	{ID: TemplateRefundRejected, Channel: "email", Subject: "Refund rejected", Body: "Hi {{name}},\n\nYour refund rejected update for {{ref}} is ready.\n\nThanks for shopping with us"},
	{ID: TemplateReturnAuthorized, Channel: "push", Subject: "Return authorized", Body: "Dear {{name}},\n\nYour return authorized update for {{ref}} is ready.\n\nThanks for shopping with us"},
	{ID: TemplateReturnReceived, Channel: "push", Subject: "Return received", Body: "Hello {{name}},\n\nYour return received update for {{ref}} is ready.\n\nThanks for shopping with us"},
	{ID: TemplateLoyaltyPointsEarned, Channel: "sms", Subject: "Loyalty points earned", Body: "Hi {{name}},\n\nYour loyalty points earned update for {{ref}} is ready.\n\nThe Shop team"},
	{ID: TemplateLoyaltyPointsRedeemed, Channel: "email", Subject: "Loyalty points redeemed", Body: "Dear {{name}},\n\nYour loyalty points redeemed update for {{ref}} is ready.\n\nThanks for shopping with us"},
	{ID: TemplateLoyaltyTierChanged, Channel: "email", Subject: "Loyalty tier changed", Body: "Hi {{name}},\n\nYour loyalty tier changed update for {{ref}} is ready.\n\nThe Shop team"},
	{ID: TemplateCouponExpiring, Channel: "email", Subject: "Coupon expiring", Body: "{{name}},\n\nYour coupon expiring update for {{ref}} is ready.\n\nThe Shop team"},
	{ID: TemplateCartAbandoned, Channel: "sms", Subject: "Cart abandoned", Body: "{{name}},\n\nYour cart abandoned update for {{ref}} is ready.\n\nThe Shop team"},
	{ID: TemplatePasswordReset, Channel: "email", Subject: "Password reset", Body: "Hi {{name}},\n\nYour password reset update for {{ref}} is ready.\n\nThanks for shopping with us"},
	{ID: TemplateAccountSuspended, Channel: "email", Subject: "Account suspended", Body: "{{name}},\n\nYour account suspended update for {{ref}} is ready.\n\nThe Shop team"},
	{ID: TemplateBackInStock, Channel: "sms", Subject: "Back in stock", Body: "Hello {{name}},\n\nYour back in stock update for {{ref}} is ready.\n\nThe Shop team"},
	{ID: TemplatePriceDrop, Channel: "email", Subject: "Price drop", Body: "Dear {{name}},\n\nYour price drop update for {{ref}} is ready.\n\nThe Shop team"},
	{ID: TemplateReviewRequest, Channel: "sms", Subject: "Review request", Body: "{{name}},\n\nYour review request update for {{ref}} is ready.\n\nThe Shop team"},
	{ID: TemplateSupplierPoApproved, Channel: "sms", Subject: "Supplier po approved", Body: "Hi {{name}},\n\nYour supplier po approved update for {{ref}} is ready.\n\nCustomer care"},
	{ID: TemplateInvoiceReady, Channel: "email", Subject: "Invoice ready", Body: "Dear {{name}},\n\nYour invoice ready update for {{ref}} is ready.\n\nThanks for shopping with us"},
	{ID: TemplateInvoiceOverdue, Channel: "push", Subject: "Invoice overdue", Body: "{{name}},\n\nYour invoice overdue update for {{ref}} is ready.\n\nCustomer care"},
	{ID: TemplateDeliverySlotBooked, Channel: "push", Subject: "Delivery slot booked", Body: "Hello {{name}},\n\nYour delivery slot booked update for {{ref}} is ready.\n\nThe Shop team"},
	{ID: TemplateDeliverySlotChanged, Channel: "push", Subject: "Delivery slot changed", Body: "Hi {{name}},\n\nYour delivery slot changed update for {{ref}} is ready.\n\nThanks for shopping with us"},
}
