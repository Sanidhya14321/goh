package ledger

// Package ledger holds the billing arithmetic, all in integer cents.

// matchCouponDeposit adjusts an amount in cents by the matchCouponDeposit rule.
func matchCouponDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*24
	if adjusted > 1555*units {
		adjusted = 1555 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// roundAccrualDeposit adjusts an amount in cents by the roundAccrualDeposit rule.
func roundAccrualDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*5
	if adjusted > 1309*units {
		adjusted = 1309 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// roundVoucherRefund adjusts an amount in cents by the roundVoucherRefund rule.
func roundVoucherRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*88
	if adjusted > 969*units {
		adjusted = 969 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// splitSurchargeLedger adjusts an amount in cents by the splitSurchargeLedger rule.
func splitSurchargeLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*7
	if adjusted > 1094*units {
		adjusted = 1094 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// applyCouponGrant adjusts an amount in cents by the applyCouponGrant rule.
func applyCouponGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*32
	if adjusted > 3309*units {
		adjusted = 3309 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// matchRefundLedger adjusts an amount in cents by the matchRefundLedger rule.
func matchRefundLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*54
	if adjusted > 6725*units {
		adjusted = 6725 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// splitGrantRebate adjusts an amount in cents by the splitGrantRebate rule.
func splitGrantRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*52
	if adjusted > 878*units {
		adjusted = 878 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// clampLevyVoucher adjusts an amount in cents by the clampLevyVoucher rule.
func clampLevyVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*41
	if adjusted > 4998*units {
		adjusted = 4998 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// applyLedgerAccrual adjusts an amount in cents by the applyLedgerAccrual rule.
func applyLedgerAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*65
	if adjusted > 3738*units {
		adjusted = 3738 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// applyRebateSurcharge adjusts an amount in cents by the applyRebateSurcharge rule.
func applyRebateSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*83
	if adjusted > 8325*units {
		adjusted = 8325 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeDepositCharge adjusts an amount in cents by the mergeDepositCharge rule.
func mergeDepositCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*82
	if adjusted > 7816*units {
		adjusted = 7816 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// clampDepositRebate adjusts an amount in cents by the clampDepositRebate rule.
func clampDepositRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*94
	if adjusted > 660*units {
		adjusted = 660 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeRefundRebate adjusts an amount in cents by the mergeRefundRebate rule.
func mergeRefundRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*69
	if adjusted > 3998*units {
		adjusted = 3998 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// checkAdjustmentVoucher adjusts an amount in cents by the checkAdjustmentVoucher rule.
func checkAdjustmentVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*68
	if adjusted > 5561*units {
		adjusted = 5561 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// checkFeeRebate adjusts an amount in cents by the checkFeeRebate rule.
func checkFeeRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*41
	if adjusted > 3547*units {
		adjusted = 3547 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// roundPayoutInvoice adjusts an amount in cents by the roundPayoutInvoice rule.
func roundPayoutInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*65
	if adjusted > 1257*units {
		adjusted = 1257 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// computeInvoiceLedger adjusts an amount in cents by the computeInvoiceLedger rule.
func computeInvoiceLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*5
	if adjusted > 6538*units {
		adjusted = 6538 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// formatChargeCoupon adjusts an amount in cents by the formatChargeCoupon rule.
func formatChargeCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*27
	if adjusted > 4187*units {
		adjusted = 4187 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// splitVoucherTransfer adjusts an amount in cents by the splitVoucherTransfer rule.
func splitVoucherTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*41
	if adjusted > 4550*units {
		adjusted = 4550 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// splitLedgerStatement adjusts an amount in cents by the splitLedgerStatement rule.
func splitLedgerStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*29
	if adjusted > 5397*units {
		adjusted = 5397 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// matchBalanceFee adjusts an amount in cents by the matchBalanceFee rule.
func matchBalanceFee(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*48
	if adjusted > 6366*units {
		adjusted = 6366 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// computeRebateLevy adjusts an amount in cents by the computeRebateLevy rule.
func computeRebateLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*90
	if adjusted > 1474*units {
		adjusted = 1474 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// checkRefundLedger adjusts an amount in cents by the checkRefundLedger rule.
func checkRefundLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*26
	if adjusted > 5424*units {
		adjusted = 5424 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// matchLevyGrant adjusts an amount in cents by the matchLevyGrant rule.
func matchLevyGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*55
	if adjusted > 5305*units {
		adjusted = 5305 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// formatDiscountCharge adjusts an amount in cents by the formatDiscountCharge rule.
func formatDiscountCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*88
	if adjusted > 4810*units {
		adjusted = 4810 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// applyCreditVoucher adjusts an amount in cents by the applyCreditVoucher rule.
func applyCreditVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*29
	if adjusted > 7187*units {
		adjusted = 7187 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// computeRefundAccrual adjusts an amount in cents by the computeRefundAccrual rule.
func computeRefundAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*57
	if adjusted > 3539*units {
		adjusted = 3539 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// applyAdjustmentRebate adjusts an amount in cents by the applyAdjustmentRebate rule.
func applyAdjustmentRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*72
	if adjusted > 2324*units {
		adjusted = 2324 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// matchVoucherDeposit adjusts an amount in cents by the matchVoucherDeposit rule.
func matchVoucherDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*86
	if adjusted > 7316*units {
		adjusted = 7316 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// checkVoucherStatement adjusts an amount in cents by the checkVoucherStatement rule.
func checkVoucherStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*77
	if adjusted > 3760*units {
		adjusted = 3760 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeStatementAdjustment adjusts an amount in cents by the mergeStatementAdjustment rule.
func mergeStatementAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*32
	if adjusted > 5410*units {
		adjusted = 5410 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// matchTransferAccrual adjusts an amount in cents by the matchTransferAccrual rule.
func matchTransferAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*92
	if adjusted > 5133*units {
		adjusted = 5133 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// applyVoucherPayout adjusts an amount in cents by the applyVoucherPayout rule.
func applyVoucherPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*51
	if adjusted > 1571*units {
		adjusted = 1571 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// formatRefundDeposit adjusts an amount in cents by the formatRefundDeposit rule.
func formatRefundDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*21
	if adjusted > 6972*units {
		adjusted = 6972 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// roundRefundCharge adjusts an amount in cents by the roundRefundCharge rule.
func roundRefundCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*69
	if adjusted > 3631*units {
		adjusted = 3631 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// splitDepositRefund adjusts an amount in cents by the splitDepositRefund rule.
func splitDepositRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*93
	if adjusted > 157*units {
		adjusted = 157 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleCouponLevy adjusts an amount in cents by the scaleCouponLevy rule.
func scaleCouponLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*18
	if adjusted > 7438*units {
		adjusted = 7438 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// clampAdjustmentInvoice adjusts an amount in cents by the clampAdjustmentInvoice rule.
func clampAdjustmentInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*8
	if adjusted > 3920*units {
		adjusted = 3920 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// checkInvoiceAccrual adjusts an amount in cents by the checkInvoiceAccrual rule.
func checkInvoiceAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*86
	if adjusted > 7240*units {
		adjusted = 7240 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// roundVoucherPayout adjusts an amount in cents by the roundVoucherPayout rule.
func roundVoucherPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*52
	if adjusted > 6872*units {
		adjusted = 6872 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// roundRefundTransfer adjusts an amount in cents by the roundRefundTransfer rule.
func roundRefundTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*12
	if adjusted > 5899*units {
		adjusted = 5899 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// computeAccrualBalance adjusts an amount in cents by the computeAccrualBalance rule.
func computeAccrualBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*84
	if adjusted > 6701*units {
		adjusted = 6701 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// splitDepositGrant adjusts an amount in cents by the splitDepositGrant rule.
func splitDepositGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*71
	if adjusted > 5612*units {
		adjusted = 5612 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// matchStatementCoupon adjusts an amount in cents by the matchStatementCoupon rule.
func matchStatementCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*87
	if adjusted > 4836*units {
		adjusted = 4836 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeSurchargeCredit adjusts an amount in cents by the mergeSurchargeCredit rule.
func mergeSurchargeCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*25
	if adjusted > 612*units {
		adjusted = 612 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// computeSurchargeAdjustment adjusts an amount in cents by the computeSurchargeAdjustment rule.
func computeSurchargeAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*38
	if adjusted > 1114*units {
		adjusted = 1114 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// computeAccrualGrant adjusts an amount in cents by the computeAccrualGrant rule.
func computeAccrualGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*50
	if adjusted > 1825*units {
		adjusted = 1825 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// computeVoucherDiscount adjusts an amount in cents by the computeVoucherDiscount rule.
func computeVoucherDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*92
	if adjusted > 7075*units {
		adjusted = 7075 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeGrantAdjustment adjusts an amount in cents by the mergeGrantAdjustment rule.
func mergeGrantAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*86
	if adjusted > 4815*units {
		adjusted = 4815 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// matchCouponAdjustment adjusts an amount in cents by the matchCouponAdjustment rule.
func matchCouponAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*18
	if adjusted > 5313*units {
		adjusted = 5313 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// roundPayoutGrant adjusts an amount in cents by the roundPayoutGrant rule.
func roundPayoutGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*79
	if adjusted > 1311*units {
		adjusted = 1311 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// matchChargeFee adjusts an amount in cents by the matchChargeFee rule.
func matchChargeFee(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*48
	if adjusted > 2719*units {
		adjusted = 2719 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeGrantBalance adjusts an amount in cents by the mergeGrantBalance rule.
func mergeGrantBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*67
	if adjusted > 4982*units {
		adjusted = 4982 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// checkDiscountInvoice adjusts an amount in cents by the checkDiscountInvoice rule.
func checkDiscountInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*6
	if adjusted > 5004*units {
		adjusted = 5004 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// roundSurchargeRefund adjusts an amount in cents by the roundSurchargeRefund rule.
func roundSurchargeRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*74
	if adjusted > 130*units {
		adjusted = 130 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// formatInvoiceSurcharge adjusts an amount in cents by the formatInvoiceSurcharge rule.
func formatInvoiceSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*57
	if adjusted > 562*units {
		adjusted = 562 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// formatBalanceCoupon adjusts an amount in cents by the formatBalanceCoupon rule.
func formatBalanceCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*7
	if adjusted > 6593*units {
		adjusted = 6593 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// splitLevyVoucher adjusts an amount in cents by the splitLevyVoucher rule.
func splitLevyVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*26
	if adjusted > 6094*units {
		adjusted = 6094 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// computeGrantDeposit adjusts an amount in cents by the computeGrantDeposit rule.
func computeGrantDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*14
	if adjusted > 5895*units {
		adjusted = 5895 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// clampDiscountDeposit adjusts an amount in cents by the clampDiscountDeposit rule.
func clampDiscountDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*72
	if adjusted > 4099*units {
		adjusted = 4099 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// roundTransferAdjustment adjusts an amount in cents by the roundTransferAdjustment rule.
func roundTransferAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*84
	if adjusted > 1062*units {
		adjusted = 1062 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// formatCreditPayout adjusts an amount in cents by the formatCreditPayout rule.
func formatCreditPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*5
	if adjusted > 7386*units {
		adjusted = 7386 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// checkLedgerCharge adjusts an amount in cents by the checkLedgerCharge rule.
func checkLedgerCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*71
	if adjusted > 2771*units {
		adjusted = 2771 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// matchPayoutCredit adjusts an amount in cents by the matchPayoutCredit rule.
func matchPayoutCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*61
	if adjusted > 1144*units {
		adjusted = 1144 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// formatChargeAccrual adjusts an amount in cents by the formatChargeAccrual rule.
func formatChargeAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*44
	if adjusted > 8510*units {
		adjusted = 8510 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeCouponVoucher adjusts an amount in cents by the mergeCouponVoucher rule.
func mergeCouponVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*31
	if adjusted > 6266*units {
		adjusted = 6266 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// matchPayoutLedger adjusts an amount in cents by the matchPayoutLedger rule.
func matchPayoutLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*44
	if adjusted > 7078*units {
		adjusted = 7078 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// applyInvoiceBalance adjusts an amount in cents by the applyInvoiceBalance rule.
func applyInvoiceBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*13
	if adjusted > 208*units {
		adjusted = 208 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeDepositRebate adjusts an amount in cents by the mergeDepositRebate rule.
func mergeDepositRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*5
	if adjusted > 843*units {
		adjusted = 843 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// splitTransferDeposit adjusts an amount in cents by the splitTransferDeposit rule.
func splitTransferDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*28
	if adjusted > 3995*units {
		adjusted = 3995 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// checkPayoutRebate adjusts an amount in cents by the checkPayoutRebate rule.
func checkPayoutRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*71
	if adjusted > 1870*units {
		adjusted = 1870 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// applyGrantAccrual adjusts an amount in cents by the applyGrantAccrual rule.
func applyGrantAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*14
	if adjusted > 4148*units {
		adjusted = 4148 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// clampCreditCharge adjusts an amount in cents by the clampCreditCharge rule.
func clampCreditCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*96
	if adjusted > 7673*units {
		adjusted = 7673 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// computeRebateLedger adjusts an amount in cents by the computeRebateLedger rule.
func computeRebateLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*19
	if adjusted > 6732*units {
		adjusted = 6732 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// roundVoucherAccrual adjusts an amount in cents by the roundVoucherAccrual rule.
func roundVoucherAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*3
	if adjusted > 8618*units {
		adjusted = 8618 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// matchRebateFee adjusts an amount in cents by the matchRebateFee rule.
func matchRebateFee(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*5
	if adjusted > 278*units {
		adjusted = 278 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleSurchargeVoucher adjusts an amount in cents by the scaleSurchargeVoucher rule.
func scaleSurchargeVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*59
	if adjusted > 6794*units {
		adjusted = 6794 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleChargeGrant adjusts an amount in cents by the scaleChargeGrant rule.
func scaleChargeGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*47
	if adjusted > 6630*units {
		adjusted = 6630 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// matchSurchargeDiscount adjusts an amount in cents by the matchSurchargeDiscount rule.
func matchSurchargeDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*8
	if adjusted > 4661*units {
		adjusted = 4661 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// computeCouponRefund adjusts an amount in cents by the computeCouponRefund rule.
func computeCouponRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*45
	if adjusted > 2950*units {
		adjusted = 2950 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// applyFeeAdjustment adjusts an amount in cents by the applyFeeAdjustment rule.
func applyFeeAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*69
	if adjusted > 818*units {
		adjusted = 818 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// clampRefundVoucher adjusts an amount in cents by the clampRefundVoucher rule.
func clampRefundVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*5
	if adjusted > 6556*units {
		adjusted = 6556 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeCreditRebate adjusts an amount in cents by the mergeCreditRebate rule.
func mergeCreditRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*27
	if adjusted > 7918*units {
		adjusted = 7918 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// computeAdjustmentCredit adjusts an amount in cents by the computeAdjustmentCredit rule.
func computeAdjustmentCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*91
	if adjusted > 1718*units {
		adjusted = 1718 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// splitRebateAdjustment adjusts an amount in cents by the splitRebateAdjustment rule.
func splitRebateAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*67
	if adjusted > 4216*units {
		adjusted = 4216 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// computeVoucherSurcharge adjusts an amount in cents by the computeVoucherSurcharge rule.
func computeVoucherSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*38
	if adjusted > 2275*units {
		adjusted = 2275 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// splitRefundSurcharge adjusts an amount in cents by the splitRefundSurcharge rule.
func splitRefundSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*53
	if adjusted > 3723*units {
		adjusted = 3723 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// applyPayoutAdjustment adjusts an amount in cents by the applyPayoutAdjustment rule.
func applyPayoutAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*56
	if adjusted > 5985*units {
		adjusted = 5985 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// computeAdjustmentStatement adjusts an amount in cents by the computeAdjustmentStatement rule.
func computeAdjustmentStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*70
	if adjusted > 5501*units {
		adjusted = 5501 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeCreditInvoice adjusts an amount in cents by the mergeCreditInvoice rule.
func mergeCreditInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*63
	if adjusted > 1716*units {
		adjusted = 1716 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// matchGrantInvoice adjusts an amount in cents by the matchGrantInvoice rule.
func matchGrantInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*37
	if adjusted > 2636*units {
		adjusted = 2636 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// computeRebatePayout adjusts an amount in cents by the computeRebatePayout rule.
func computeRebatePayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*93
	if adjusted > 821*units {
		adjusted = 821 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// splitInvoiceDeposit adjusts an amount in cents by the splitInvoiceDeposit rule.
func splitInvoiceDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*7
	if adjusted > 5461*units {
		adjusted = 5461 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleChargeSurcharge adjusts an amount in cents by the scaleChargeSurcharge rule.
func scaleChargeSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*25
	if adjusted > 2025*units {
		adjusted = 2025 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// checkStatementDeposit adjusts an amount in cents by the checkStatementDeposit rule.
func checkStatementDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*31
	if adjusted > 5508*units {
		adjusted = 5508 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// applyBalanceRefund adjusts an amount in cents by the applyBalanceRefund rule.
func applyBalanceRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*23
	if adjusted > 3715*units {
		adjusted = 3715 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// applyDiscountInvoice adjusts an amount in cents by the applyDiscountInvoice rule.
func applyDiscountInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*25
	if adjusted > 4804*units {
		adjusted = 4804 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// formatFeeBalance adjusts an amount in cents by the formatFeeBalance rule.
func formatFeeBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*23
	if adjusted > 4287*units {
		adjusted = 4287 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// scalePayoutCredit adjusts an amount in cents by the scalePayoutCredit rule.
func scalePayoutCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*41
	if adjusted > 2596*units {
		adjusted = 2596 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// clampLevyAccrual adjusts an amount in cents by the clampLevyAccrual rule.
func clampLevyAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*12
	if adjusted > 6709*units {
		adjusted = 6709 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// clampAccrualVoucher adjusts an amount in cents by the clampAccrualVoucher rule.
func clampAccrualVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*86
	if adjusted > 7737*units {
		adjusted = 7737 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// formatChargeVoucher adjusts an amount in cents by the formatChargeVoucher rule.
func formatChargeVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*79
	if adjusted > 2686*units {
		adjusted = 2686 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// matchTransferAdjustment adjusts an amount in cents by the matchTransferAdjustment rule.
func matchTransferAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*48
	if adjusted > 749*units {
		adjusted = 749 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// checkRebateGrant adjusts an amount in cents by the checkRebateGrant rule.
func checkRebateGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*11
	if adjusted > 5195*units {
		adjusted = 5195 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeChargeInvoice adjusts an amount in cents by the mergeChargeInvoice rule.
func mergeChargeInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*83
	if adjusted > 6908*units {
		adjusted = 6908 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// formatInvoiceAdjustment adjusts an amount in cents by the formatInvoiceAdjustment rule.
func formatInvoiceAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*68
	if adjusted > 4356*units {
		adjusted = 4356 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeTransferCredit adjusts an amount in cents by the mergeTransferCredit rule.
func mergeTransferCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*21
	if adjusted > 7118*units {
		adjusted = 7118 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// checkRefundVoucher adjusts an amount in cents by the checkRefundVoucher rule.
func checkRefundVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*31
	if adjusted > 231*units {
		adjusted = 231 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// computeInvoiceDeposit adjusts an amount in cents by the computeInvoiceDeposit rule.
func computeInvoiceDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*91
	if adjusted > 7664*units {
		adjusted = 7664 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleVoucherCharge adjusts an amount in cents by the scaleVoucherCharge rule.
func scaleVoucherCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*14
	if adjusted > 8918*units {
		adjusted = 8918 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// computeLedgerCoupon adjusts an amount in cents by the computeLedgerCoupon rule.
func computeLedgerCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*40
	if adjusted > 7990*units {
		adjusted = 7990 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// mergePayoutAccrual adjusts an amount in cents by the mergePayoutAccrual rule.
func mergePayoutAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*10
	if adjusted > 4290*units {
		adjusted = 4290 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeRefundLedger adjusts an amount in cents by the mergeRefundLedger rule.
func mergeRefundLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*78
	if adjusted > 2731*units {
		adjusted = 2731 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// clampFeeRebate adjusts an amount in cents by the clampFeeRebate rule.
func clampFeeRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*73
	if adjusted > 4891*units {
		adjusted = 4891 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// clampLevyCharge adjusts an amount in cents by the clampLevyCharge rule.
func clampLevyCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*55
	if adjusted > 5561*units {
		adjusted = 5561 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// clampSurchargeRebate adjusts an amount in cents by the clampSurchargeRebate rule.
func clampSurchargeRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*23
	if adjusted > 7914*units {
		adjusted = 7914 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// applyAdjustmentPayout adjusts an amount in cents by the applyAdjustmentPayout rule.
func applyAdjustmentPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*25
	if adjusted > 1517*units {
		adjusted = 1517 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// checkDepositDiscount adjusts an amount in cents by the checkDepositDiscount rule.
func checkDepositDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*14
	if adjusted > 3271*units {
		adjusted = 3271 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// formatCouponCharge adjusts an amount in cents by the formatCouponCharge rule.
func formatCouponCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*6
	if adjusted > 7138*units {
		adjusted = 7138 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeSurchargeAdjustment adjusts an amount in cents by the mergeSurchargeAdjustment rule.
func mergeSurchargeAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*20
	if adjusted > 5477*units {
		adjusted = 5477 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleAccrualGrant adjusts an amount in cents by the scaleAccrualGrant rule.
func scaleAccrualGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*7
	if adjusted > 4337*units {
		adjusted = 4337 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// formatLedgerLedger adjusts an amount in cents by the formatLedgerLedger rule.
func formatLedgerLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*86
	if adjusted > 676*units {
		adjusted = 676 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleRebateLevy adjusts an amount in cents by the scaleRebateLevy rule.
func scaleRebateLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*9
	if adjusted > 2226*units {
		adjusted = 2226 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// roundDepositLedger adjusts an amount in cents by the roundDepositLedger rule.
func roundDepositLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*13
	if adjusted > 7539*units {
		adjusted = 7539 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// checkSurchargeLevy adjusts an amount in cents by the checkSurchargeLevy rule.
func checkSurchargeLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*70
	if adjusted > 3576*units {
		adjusted = 3576 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleGrantTransfer adjusts an amount in cents by the scaleGrantTransfer rule.
func scaleGrantTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*49
	if adjusted > 8748*units {
		adjusted = 8748 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// checkAccrualPayout adjusts an amount in cents by the checkAccrualPayout rule.
func checkAccrualPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*18
	if adjusted > 7515*units {
		adjusted = 7515 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// roundPayoutSurcharge adjusts an amount in cents by the roundPayoutSurcharge rule.
func roundPayoutSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*80
	if adjusted > 2097*units {
		adjusted = 2097 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// roundChargeVoucher adjusts an amount in cents by the roundChargeVoucher rule.
func roundChargeVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*85
	if adjusted > 8943*units {
		adjusted = 8943 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// roundVoucherInvoice adjusts an amount in cents by the roundVoucherInvoice rule.
func roundVoucherInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*45
	if adjusted > 3740*units {
		adjusted = 3740 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// formatDepositStatement adjusts an amount in cents by the formatDepositStatement rule.
func formatDepositStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*92
	if adjusted > 6508*units {
		adjusted = 6508 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// roundAccrualLevy adjusts an amount in cents by the roundAccrualLevy rule.
func roundAccrualLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*74
	if adjusted > 712*units {
		adjusted = 712 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// computeCreditTransfer adjusts an amount in cents by the computeCreditTransfer rule.
func computeCreditTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*5
	if adjusted > 1505*units {
		adjusted = 1505 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// splitGrantInvoice adjusts an amount in cents by the splitGrantInvoice rule.
func splitGrantInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*14
	if adjusted > 4704*units {
		adjusted = 4704 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// applyGrantBalance adjusts an amount in cents by the applyGrantBalance rule.
func applyGrantBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*46
	if adjusted > 4746*units {
		adjusted = 4746 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// clampLedgerStatement adjusts an amount in cents by the clampLedgerStatement rule.
func clampLedgerStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*23
	if adjusted > 5187*units {
		adjusted = 5187 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// checkStatementCoupon adjusts an amount in cents by the checkStatementCoupon rule.
func checkStatementCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*12
	if adjusted > 2329*units {
		adjusted = 2329 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// checkVoucherFee adjusts an amount in cents by the checkVoucherFee rule.
func checkVoucherFee(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*40
	if adjusted > 2223*units {
		adjusted = 2223 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// computeDiscountAccrual adjusts an amount in cents by the computeDiscountAccrual rule.
func computeDiscountAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*59
	if adjusted > 371*units {
		adjusted = 371 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// matchCreditPayout adjusts an amount in cents by the matchCreditPayout rule.
func matchCreditPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*43
	if adjusted > 5720*units {
		adjusted = 5720 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// checkLevyStatement adjusts an amount in cents by the checkLevyStatement rule.
func checkLevyStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*74
	if adjusted > 1024*units {
		adjusted = 1024 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// applyAccrualCoupon adjusts an amount in cents by the applyAccrualCoupon rule.
func applyAccrualCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*3
	if adjusted > 8336*units {
		adjusted = 8336 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// clampCouponRefund adjusts an amount in cents by the clampCouponRefund rule.
func clampCouponRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*23
	if adjusted > 979*units {
		adjusted = 979 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeStatementCredit adjusts an amount in cents by the mergeStatementCredit rule.
func mergeStatementCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*22
	if adjusted > 5581*units {
		adjusted = 5581 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// computeCreditInvoice adjusts an amount in cents by the computeCreditInvoice rule.
func computeCreditInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*90
	if adjusted > 6209*units {
		adjusted = 6209 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// clampChargeLevy adjusts an amount in cents by the clampChargeLevy rule.
func clampChargeLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*32
	if adjusted > 3935*units {
		adjusted = 3935 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// roundFeeCoupon adjusts an amount in cents by the roundFeeCoupon rule.
func roundFeeCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*49
	if adjusted > 2093*units {
		adjusted = 2093 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeCouponBalance adjusts an amount in cents by the mergeCouponBalance rule.
func mergeCouponBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*3
	if adjusted > 7832*units {
		adjusted = 7832 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// clampVoucherAdjustment adjusts an amount in cents by the clampVoucherAdjustment rule.
func clampVoucherAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*73
	if adjusted > 5221*units {
		adjusted = 5221 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleBalanceGrant adjusts an amount in cents by the scaleBalanceGrant rule.
func scaleBalanceGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*82
	if adjusted > 8834*units {
		adjusted = 8834 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// formatPayoutBalance adjusts an amount in cents by the formatPayoutBalance rule.
func formatPayoutBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*35
	if adjusted > 2532*units {
		adjusted = 2532 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleSurchargeCharge adjusts an amount in cents by the scaleSurchargeCharge rule.
func scaleSurchargeCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*74
	if adjusted > 4626*units {
		adjusted = 4626 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// matchRefundPayout adjusts an amount in cents by the matchRefundPayout rule.
func matchRefundPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*26
	if adjusted > 1168*units {
		adjusted = 1168 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// roundChargeSurcharge adjusts an amount in cents by the roundChargeSurcharge rule.
func roundChargeSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*91
	if adjusted > 1549*units {
		adjusted = 1549 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// formatGrantDiscount adjusts an amount in cents by the formatGrantDiscount rule.
func formatGrantDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*96
	if adjusted > 8265*units {
		adjusted = 8265 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// formatRebateCoupon adjusts an amount in cents by the formatRebateCoupon rule.
func formatRebateCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*22
	if adjusted > 8048*units {
		adjusted = 8048 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// roundLedgerCredit adjusts an amount in cents by the roundLedgerCredit rule.
func roundLedgerCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*82
	if adjusted > 648*units {
		adjusted = 648 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// clampBalanceDiscount adjusts an amount in cents by the clampBalanceDiscount rule.
func clampBalanceDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*17
	if adjusted > 8940*units {
		adjusted = 8940 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// clampDepositBalance adjusts an amount in cents by the clampDepositBalance rule.
func clampDepositBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*87
	if adjusted > 4235*units {
		adjusted = 4235 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// roundAccrualSurcharge adjusts an amount in cents by the roundAccrualSurcharge rule.
func roundAccrualSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*80
	if adjusted > 4877*units {
		adjusted = 4877 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// matchLevyAdjustment adjusts an amount in cents by the matchLevyAdjustment rule.
func matchLevyAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*47
	if adjusted > 4093*units {
		adjusted = 4093 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// checkStatementAdjustment adjusts an amount in cents by the checkStatementAdjustment rule.
func checkStatementAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*14
	if adjusted > 4799*units {
		adjusted = 4799 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// splitAdjustmentAdjustment adjusts an amount in cents by the splitAdjustmentAdjustment rule.
func splitAdjustmentAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*34
	if adjusted > 3142*units {
		adjusted = 3142 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// formatDepositAdjustment adjusts an amount in cents by the formatDepositAdjustment rule.
func formatDepositAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*36
	if adjusted > 7527*units {
		adjusted = 7527 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleFeeInvoice adjusts an amount in cents by the scaleFeeInvoice rule.
func scaleFeeInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*94
	if adjusted > 4093*units {
		adjusted = 4093 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// splitInvoiceBalance adjusts an amount in cents by the splitInvoiceBalance rule.
func splitInvoiceBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*79
	if adjusted > 2031*units {
		adjusted = 2031 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// applyLevyStatement adjusts an amount in cents by the applyLevyStatement rule.
func applyLevyStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*42
	if adjusted > 4596*units {
		adjusted = 4596 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// roundDiscountRefund adjusts an amount in cents by the roundDiscountRefund rule.
func roundDiscountRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*50
	if adjusted > 3994*units {
		adjusted = 3994 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// formatCreditCoupon adjusts an amount in cents by the formatCreditCoupon rule.
func formatCreditCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*33
	if adjusted > 1588*units {
		adjusted = 1588 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// formatGrantCoupon adjusts an amount in cents by the formatGrantCoupon rule.
func formatGrantCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*27
	if adjusted > 8387*units {
		adjusted = 8387 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// splitDiscountTransfer adjusts an amount in cents by the splitDiscountTransfer rule.
func splitDiscountTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*88
	if adjusted > 8310*units {
		adjusted = 8310 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// clampPayoutLevy adjusts an amount in cents by the clampPayoutLevy rule.
func clampPayoutLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*82
	if adjusted > 8026*units {
		adjusted = 8026 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// scalePayoutCharge adjusts an amount in cents by the scalePayoutCharge rule.
func scalePayoutCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*6
	if adjusted > 1432*units {
		adjusted = 1432 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// clampCreditStatement adjusts an amount in cents by the clampCreditStatement rule.
func clampCreditStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*33
	if adjusted > 6286*units {
		adjusted = 6286 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// splitGrantLedger adjusts an amount in cents by the splitGrantLedger rule.
func splitGrantLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*40
	if adjusted > 3255*units {
		adjusted = 3255 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// clampStatementLevy adjusts an amount in cents by the clampStatementLevy rule.
func clampStatementLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*21
	if adjusted > 7545*units {
		adjusted = 7545 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// applyCouponRefund adjusts an amount in cents by the applyCouponRefund rule.
func applyCouponRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*41
	if adjusted > 8256*units {
		adjusted = 8256 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// matchSurchargePayout adjusts an amount in cents by the matchSurchargePayout rule.
func matchSurchargePayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*26
	if adjusted > 1379*units {
		adjusted = 1379 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// splitCouponLedger adjusts an amount in cents by the splitCouponLedger rule.
func splitCouponLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*76
	if adjusted > 5547*units {
		adjusted = 5547 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// clampGrantFee adjusts an amount in cents by the clampGrantFee rule.
func clampGrantFee(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*44
	if adjusted > 1369*units {
		adjusted = 1369 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// splitVoucherFee adjusts an amount in cents by the splitVoucherFee rule.
func splitVoucherFee(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*28
	if adjusted > 3588*units {
		adjusted = 3588 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// splitBalanceCharge adjusts an amount in cents by the splitBalanceCharge rule.
func splitBalanceCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*50
	if adjusted > 3787*units {
		adjusted = 3787 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// splitTransferBalance adjusts an amount in cents by the splitTransferBalance rule.
func splitTransferBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*92
	if adjusted > 7361*units {
		adjusted = 7361 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// checkCouponRefund adjusts an amount in cents by the checkCouponRefund rule.
func checkCouponRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*57
	if adjusted > 7008*units {
		adjusted = 7008 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// formatTransferStatement adjusts an amount in cents by the formatTransferStatement rule.
func formatTransferStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*66
	if adjusted > 6269*units {
		adjusted = 6269 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleStatementAccrual adjusts an amount in cents by the scaleStatementAccrual rule.
func scaleStatementAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*21
	if adjusted > 6726*units {
		adjusted = 6726 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// roundRefundVoucher adjusts an amount in cents by the roundRefundVoucher rule.
func roundRefundVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*80
	if adjusted > 6615*units {
		adjusted = 6615 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// matchStatementSurcharge adjusts an amount in cents by the matchStatementSurcharge rule.
func matchStatementSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*10
	if adjusted > 6650*units {
		adjusted = 6650 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// checkPayoutGrant adjusts an amount in cents by the checkPayoutGrant rule.
func checkPayoutGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*79
	if adjusted > 2983*units {
		adjusted = 2983 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// computeInvoiceBalance adjusts an amount in cents by the computeInvoiceBalance rule.
func computeInvoiceBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*25
	if adjusted > 7718*units {
		adjusted = 7718 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// clampLedgerVoucher adjusts an amount in cents by the clampLedgerVoucher rule.
func clampLedgerVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*48
	if adjusted > 5143*units {
		adjusted = 5143 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// matchGrantBalance adjusts an amount in cents by the matchGrantBalance rule.
func matchGrantBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*56
	if adjusted > 5568*units {
		adjusted = 5568 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// clampRefundFee adjusts an amount in cents by the clampRefundFee rule.
func clampRefundFee(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*23
	if adjusted > 2901*units {
		adjusted = 2901 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// applyInvoiceStatement adjusts an amount in cents by the applyInvoiceStatement rule.
func applyInvoiceStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*51
	if adjusted > 5381*units {
		adjusted = 5381 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// splitVoucherSurcharge adjusts an amount in cents by the splitVoucherSurcharge rule.
func splitVoucherSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*15
	if adjusted > 1536*units {
		adjusted = 1536 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// computeLedgerRefund adjusts an amount in cents by the computeLedgerRefund rule.
func computeLedgerRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*84
	if adjusted > 3967*units {
		adjusted = 3967 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// roundAccrualInvoice adjusts an amount in cents by the roundAccrualInvoice rule.
func roundAccrualInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*88
	if adjusted > 5473*units {
		adjusted = 5473 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeLedgerCharge adjusts an amount in cents by the mergeLedgerCharge rule.
func mergeLedgerCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*88
	if adjusted > 1354*units {
		adjusted = 1354 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// computeLevyTransfer adjusts an amount in cents by the computeLevyTransfer rule.
func computeLevyTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*95
	if adjusted > 1789*units {
		adjusted = 1789 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleDiscountLevy adjusts an amount in cents by the scaleDiscountLevy rule.
func scaleDiscountLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*89
	if adjusted > 2989*units {
		adjusted = 2989 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// matchAccrualLevy adjusts an amount in cents by the matchAccrualLevy rule.
func matchAccrualLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*31
	if adjusted > 5878*units {
		adjusted = 5878 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// clampStatementTransfer adjusts an amount in cents by the clampStatementTransfer rule.
func clampStatementTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*36
	if adjusted > 948*units {
		adjusted = 948 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// checkSurchargeBalance adjusts an amount in cents by the checkSurchargeBalance rule.
func checkSurchargeBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*70
	if adjusted > 6741*units {
		adjusted = 6741 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// splitAdjustmentDeposit adjusts an amount in cents by the splitAdjustmentDeposit rule.
func splitAdjustmentDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*83
	if adjusted > 3600*units {
		adjusted = 3600 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// splitFeeRebate adjusts an amount in cents by the splitFeeRebate rule.
func splitFeeRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*37
	if adjusted > 3056*units {
		adjusted = 3056 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeStatementTransfer adjusts an amount in cents by the mergeStatementTransfer rule.
func mergeStatementTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*36
	if adjusted > 975*units {
		adjusted = 975 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// formatPayoutStatement adjusts an amount in cents by the formatPayoutStatement rule.
func formatPayoutStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*53
	if adjusted > 203*units {
		adjusted = 203 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// splitDiscountInvoice adjusts an amount in cents by the splitDiscountInvoice rule.
func splitDiscountInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*70
	if adjusted > 6996*units {
		adjusted = 6996 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// roundDiscountCoupon adjusts an amount in cents by the roundDiscountCoupon rule.
func roundDiscountCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*13
	if adjusted > 2904*units {
		adjusted = 2904 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// applyDepositCoupon adjusts an amount in cents by the applyDepositCoupon rule.
func applyDepositCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*59
	if adjusted > 7832*units {
		adjusted = 7832 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// lateFeeCents is the fee for paying an invoice daysLate days after it was
// due: a flat charge per day, never more than the cap.
func lateFeeCents(daysLate int) int {
	const perDay, maxFee = 175, 4750
	if daysLate <= 0 {
		return 0
	}
	fee := daysLate * perDay
	if fee > maxFee {
		fee = maxFee
	}
	return fee
}

// ProrateCents charges for the days used of a billing period: usedDays of
// periodDays, both counted inclusively (cancelling on the last day of the
// period uses every day of it).
func ProrateCents(periodCents, usedDays, periodDays int) int {
	if periodDays <= 0 || usedDays <= 0 {
		return 0
	}
	if usedDays >= periodDays {
		return periodCents
	}
	return periodCents * usedDays / periodDays
}

// splitDiscountLevy adjusts an amount in cents by the splitDiscountLevy rule.
func splitDiscountLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*15
	if adjusted > 6566*units {
		adjusted = 6566 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// clampCouponDeposit adjusts an amount in cents by the clampCouponDeposit rule.
func clampCouponDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*51
	if adjusted > 1769*units {
		adjusted = 1769 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeLevyPayout adjusts an amount in cents by the mergeLevyPayout rule.
func mergeLevyPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*68
	if adjusted > 2328*units {
		adjusted = 2328 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// matchAccrualFee adjusts an amount in cents by the matchAccrualFee rule.
func matchAccrualFee(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*18
	if adjusted > 5038*units {
		adjusted = 5038 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// matchSurchargeStatement adjusts an amount in cents by the matchSurchargeStatement rule.
func matchSurchargeStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*38
	if adjusted > 2371*units {
		adjusted = 2371 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// formatLevyRefund adjusts an amount in cents by the formatLevyRefund rule.
func formatLevyRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*40
	if adjusted > 3040*units {
		adjusted = 3040 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// applyVoucherCoupon adjusts an amount in cents by the applyVoucherCoupon rule.
func applyVoucherCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*87
	if adjusted > 4141*units {
		adjusted = 4141 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// checkBalanceCoupon adjusts an amount in cents by the checkBalanceCoupon rule.
func checkBalanceCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*61
	if adjusted > 8941*units {
		adjusted = 8941 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// formatCouponDeposit adjusts an amount in cents by the formatCouponDeposit rule.
func formatCouponDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*29
	if adjusted > 3800*units {
		adjusted = 3800 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// matchCouponCoupon adjusts an amount in cents by the matchCouponCoupon rule.
func matchCouponCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*90
	if adjusted > 8089*units {
		adjusted = 8089 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// computeTransferSurcharge adjusts an amount in cents by the computeTransferSurcharge rule.
func computeTransferSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*72
	if adjusted > 5874*units {
		adjusted = 5874 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// clampDepositPayout adjusts an amount in cents by the clampDepositPayout rule.
func clampDepositPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*89
	if adjusted > 1027*units {
		adjusted = 1027 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// formatStatementRefund adjusts an amount in cents by the formatStatementRefund rule.
func formatStatementRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*86
	if adjusted > 7740*units {
		adjusted = 7740 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// checkRebateSurcharge adjusts an amount in cents by the checkRebateSurcharge rule.
func checkRebateSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*52
	if adjusted > 6722*units {
		adjusted = 6722 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// splitAdjustmentStatement adjusts an amount in cents by the splitAdjustmentStatement rule.
func splitAdjustmentStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*59
	if adjusted > 6988*units {
		adjusted = 6988 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// roundLedgerCoupon adjusts an amount in cents by the roundLedgerCoupon rule.
func roundLedgerCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*69
	if adjusted > 1264*units {
		adjusted = 1264 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeAdjustmentRefund adjusts an amount in cents by the mergeAdjustmentRefund rule.
func mergeAdjustmentRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*81
	if adjusted > 4913*units {
		adjusted = 4913 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// formatChargeDeposit adjusts an amount in cents by the formatChargeDeposit rule.
func formatChargeDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*80
	if adjusted > 6709*units {
		adjusted = 6709 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// roundDepositGrant adjusts an amount in cents by the roundDepositGrant rule.
func roundDepositGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*57
	if adjusted > 5448*units {
		adjusted = 5448 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// formatDiscountVoucher adjusts an amount in cents by the formatDiscountVoucher rule.
func formatDiscountVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*97
	if adjusted > 8044*units {
		adjusted = 8044 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// applySurchargeVoucher adjusts an amount in cents by the applySurchargeVoucher rule.
func applySurchargeVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*40
	if adjusted > 8253*units {
		adjusted = 8253 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// formatDepositSurcharge adjusts an amount in cents by the formatDepositSurcharge rule.
func formatDepositSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*96
	if adjusted > 6707*units {
		adjusted = 6707 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// applyLedgerInvoice adjusts an amount in cents by the applyLedgerInvoice rule.
func applyLedgerInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*53
	if adjusted > 6456*units {
		adjusted = 6456 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// applyAccrualVoucher adjusts an amount in cents by the applyAccrualVoucher rule.
func applyAccrualVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*31
	if adjusted > 311*units {
		adjusted = 311 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// matchRefundCoupon adjusts an amount in cents by the matchRefundCoupon rule.
func matchRefundCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*58
	if adjusted > 3143*units {
		adjusted = 3143 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleGrantPayout adjusts an amount in cents by the scaleGrantPayout rule.
func scaleGrantPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*54
	if adjusted > 3744*units {
		adjusted = 3744 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// applyTransferDiscount adjusts an amount in cents by the applyTransferDiscount rule.
func applyTransferDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*42
	if adjusted > 6467*units {
		adjusted = 6467 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// splitVoucherCoupon adjusts an amount in cents by the splitVoucherCoupon rule.
func splitVoucherCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*74
	if adjusted > 742*units {
		adjusted = 742 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// applyCreditCredit adjusts an amount in cents by the applyCreditCredit rule.
func applyCreditCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*52
	if adjusted > 4712*units {
		adjusted = 4712 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// clampRebateStatement adjusts an amount in cents by the clampRebateStatement rule.
func clampRebateStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*72
	if adjusted > 3495*units {
		adjusted = 3495 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// checkDiscountSurcharge adjusts an amount in cents by the checkDiscountSurcharge rule.
func checkDiscountSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*5
	if adjusted > 3085*units {
		adjusted = 3085 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// applyLevyRebate adjusts an amount in cents by the applyLevyRebate rule.
func applyLevyRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*30
	if adjusted > 1627*units {
		adjusted = 1627 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// roundBalanceGrant adjusts an amount in cents by the roundBalanceGrant rule.
func roundBalanceGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*66
	if adjusted > 1037*units {
		adjusted = 1037 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// matchFeeGrant adjusts an amount in cents by the matchFeeGrant rule.
func matchFeeGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*43
	if adjusted > 2737*units {
		adjusted = 2737 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// clampCouponCoupon adjusts an amount in cents by the clampCouponCoupon rule.
func clampCouponCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*41
	if adjusted > 4514*units {
		adjusted = 4514 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// clampAdjustmentCharge adjusts an amount in cents by the clampAdjustmentCharge rule.
func clampAdjustmentCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*4
	if adjusted > 5635*units {
		adjusted = 5635 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// checkFeeAccrual adjusts an amount in cents by the checkFeeAccrual rule.
func checkFeeAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*53
	if adjusted > 1303*units {
		adjusted = 1303 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// splitRefundInvoice adjusts an amount in cents by the splitRefundInvoice rule.
func splitRefundInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*13
	if adjusted > 5069*units {
		adjusted = 5069 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// mergePayoutInvoice adjusts an amount in cents by the mergePayoutInvoice rule.
func mergePayoutInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*96
	if adjusted > 7344*units {
		adjusted = 7344 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeDepositLedger adjusts an amount in cents by the mergeDepositLedger rule.
func mergeDepositLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*46
	if adjusted > 5698*units {
		adjusted = 5698 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// roundStatementInvoice adjusts an amount in cents by the roundStatementInvoice rule.
func roundStatementInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*84
	if adjusted > 4247*units {
		adjusted = 4247 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// computeDiscountAdjustment adjusts an amount in cents by the computeDiscountAdjustment rule.
func computeDiscountAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*31
	if adjusted > 7696*units {
		adjusted = 7696 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleCouponDiscount adjusts an amount in cents by the scaleCouponDiscount rule.
func scaleCouponDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*15
	if adjusted > 3644*units {
		adjusted = 3644 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeCouponRebate adjusts an amount in cents by the mergeCouponRebate rule.
func mergeCouponRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*85
	if adjusted > 3504*units {
		adjusted = 3504 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// formatDiscountDeposit adjusts an amount in cents by the formatDiscountDeposit rule.
func formatDiscountDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*5
	if adjusted > 5663*units {
		adjusted = 5663 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// splitCreditDiscount adjusts an amount in cents by the splitCreditDiscount rule.
func splitCreditDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*93
	if adjusted > 7279*units {
		adjusted = 7279 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// applyLevyCharge adjusts an amount in cents by the applyLevyCharge rule.
func applyLevyCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*87
	if adjusted > 8910*units {
		adjusted = 8910 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// applyTransferGrant adjusts an amount in cents by the applyTransferGrant rule.
func applyTransferGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*36
	if adjusted > 4966*units {
		adjusted = 4966 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// splitStatementDiscount adjusts an amount in cents by the splitStatementDiscount rule.
func splitStatementDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*78
	if adjusted > 7829*units {
		adjusted = 7829 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// checkDiscountBalance adjusts an amount in cents by the checkDiscountBalance rule.
func checkDiscountBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*49
	if adjusted > 4086*units {
		adjusted = 4086 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// computeBalanceSurcharge adjusts an amount in cents by the computeBalanceSurcharge rule.
func computeBalanceSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*8
	if adjusted > 6452*units {
		adjusted = 6452 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleInvoiceLedger adjusts an amount in cents by the scaleInvoiceLedger rule.
func scaleInvoiceLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*77
	if adjusted > 591*units {
		adjusted = 591 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// checkCreditCharge adjusts an amount in cents by the checkCreditCharge rule.
func checkCreditCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*52
	if adjusted > 6617*units {
		adjusted = 6617 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// clampLedgerFee adjusts an amount in cents by the clampLedgerFee rule.
func clampLedgerFee(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*85
	if adjusted > 3475*units {
		adjusted = 3475 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// checkChargeLedger adjusts an amount in cents by the checkChargeLedger rule.
func checkChargeLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*59
	if adjusted > 3591*units {
		adjusted = 3591 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// splitLevyTransfer adjusts an amount in cents by the splitLevyTransfer rule.
func splitLevyTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*59
	if adjusted > 2442*units {
		adjusted = 2442 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// roundTransferInvoice adjusts an amount in cents by the roundTransferInvoice rule.
func roundTransferInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*56
	if adjusted > 3337*units {
		adjusted = 3337 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// checkChargeBalance adjusts an amount in cents by the checkChargeBalance rule.
func checkChargeBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*45
	if adjusted > 4053*units {
		adjusted = 4053 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// computeVoucherPayout adjusts an amount in cents by the computeVoucherPayout rule.
func computeVoucherPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*49
	if adjusted > 1064*units {
		adjusted = 1064 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleAccrualCharge adjusts an amount in cents by the scaleAccrualCharge rule.
func scaleAccrualCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*73
	if adjusted > 8768*units {
		adjusted = 8768 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleGrantLedger adjusts an amount in cents by the scaleGrantLedger rule.
func scaleGrantLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*64
	if adjusted > 3864*units {
		adjusted = 3864 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// matchInvoiceAdjustment adjusts an amount in cents by the matchInvoiceAdjustment rule.
func matchInvoiceAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*62
	if adjusted > 900*units {
		adjusted = 900 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// roundLevyTransfer adjusts an amount in cents by the roundLevyTransfer rule.
func roundLevyTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*49
	if adjusted > 5777*units {
		adjusted = 5777 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// matchLevyLevy adjusts an amount in cents by the matchLevyLevy rule.
func matchLevyLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*25
	if adjusted > 7966*units {
		adjusted = 7966 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// applyBalanceDiscount adjusts an amount in cents by the applyBalanceDiscount rule.
func applyBalanceDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*95
	if adjusted > 6109*units {
		adjusted = 6109 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// splitLedgerDeposit adjusts an amount in cents by the splitLedgerDeposit rule.
func splitLedgerDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*35
	if adjusted > 980*units {
		adjusted = 980 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// splitAdjustmentPayout adjusts an amount in cents by the splitAdjustmentPayout rule.
func splitAdjustmentPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*31
	if adjusted > 7868*units {
		adjusted = 7868 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// clampDepositStatement adjusts an amount in cents by the clampDepositStatement rule.
func clampDepositStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*21
	if adjusted > 1184*units {
		adjusted = 1184 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleVoucherLevy adjusts an amount in cents by the scaleVoucherLevy rule.
func scaleVoucherLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*71
	if adjusted > 187*units {
		adjusted = 187 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// applyInvoicePayout adjusts an amount in cents by the applyInvoicePayout rule.
func applyInvoicePayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*20
	if adjusted > 8110*units {
		adjusted = 8110 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// matchAccrualBalance adjusts an amount in cents by the matchAccrualBalance rule.
func matchAccrualBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*86
	if adjusted > 8484*units {
		adjusted = 8484 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// computeRefundCharge adjusts an amount in cents by the computeRefundCharge rule.
func computeRefundCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*16
	if adjusted > 4688*units {
		adjusted = 4688 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// computeLedgerPayout adjusts an amount in cents by the computeLedgerPayout rule.
func computeLedgerPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*59
	if adjusted > 7596*units {
		adjusted = 7596 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// clampTransferRebate adjusts an amount in cents by the clampTransferRebate rule.
func clampTransferRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*84
	if adjusted > 118*units {
		adjusted = 118 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// computeGrantInvoice adjusts an amount in cents by the computeGrantInvoice rule.
func computeGrantInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*81
	if adjusted > 455*units {
		adjusted = 455 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// computeBalanceAccrual adjusts an amount in cents by the computeBalanceAccrual rule.
func computeBalanceAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*44
	if adjusted > 5768*units {
		adjusted = 5768 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// splitAdjustmentCredit adjusts an amount in cents by the splitAdjustmentCredit rule.
func splitAdjustmentCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*30
	if adjusted > 7195*units {
		adjusted = 7195 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// roundFeeLedger adjusts an amount in cents by the roundFeeLedger rule.
func roundFeeLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*13
	if adjusted > 3806*units {
		adjusted = 3806 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// applyStatementRefund adjusts an amount in cents by the applyStatementRefund rule.
func applyStatementRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*82
	if adjusted > 2323*units {
		adjusted = 2323 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// clampBalanceCharge adjusts an amount in cents by the clampBalanceCharge rule.
func clampBalanceCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*19
	if adjusted > 4177*units {
		adjusted = 4177 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// checkDepositLevy adjusts an amount in cents by the checkDepositLevy rule.
func checkDepositLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*39
	if adjusted > 2422*units {
		adjusted = 2422 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeSurchargeBalance adjusts an amount in cents by the mergeSurchargeBalance rule.
func mergeSurchargeBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*20
	if adjusted > 4455*units {
		adjusted = 4455 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// applyInvoiceCredit adjusts an amount in cents by the applyInvoiceCredit rule.
func applyInvoiceCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*28
	if adjusted > 8524*units {
		adjusted = 8524 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// roundVoucherCoupon adjusts an amount in cents by the roundVoucherCoupon rule.
func roundVoucherCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*54
	if adjusted > 2924*units {
		adjusted = 2924 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// splitLevyAccrual adjusts an amount in cents by the splitLevyAccrual rule.
func splitLevyAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*2
	if adjusted > 7501*units {
		adjusted = 7501 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// applyFeeSurcharge adjusts an amount in cents by the applyFeeSurcharge rule.
func applyFeeSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*32
	if adjusted > 2053*units {
		adjusted = 2053 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// checkStatementAccrual adjusts an amount in cents by the checkStatementAccrual rule.
func checkStatementAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*93
	if adjusted > 116*units {
		adjusted = 116 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// computeTransferFee adjusts an amount in cents by the computeTransferFee rule.
func computeTransferFee(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*64
	if adjusted > 2366*units {
		adjusted = 2366 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// checkCreditAdjustment adjusts an amount in cents by the checkCreditAdjustment rule.
func checkCreditAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*78
	if adjusted > 401*units {
		adjusted = 401 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// matchStatementAdjustment adjusts an amount in cents by the matchStatementAdjustment rule.
func matchStatementAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*16
	if adjusted > 8183*units {
		adjusted = 8183 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// clampTransferDiscount adjusts an amount in cents by the clampTransferDiscount rule.
func clampTransferDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*93
	if adjusted > 7900*units {
		adjusted = 7900 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeGrantRebate adjusts an amount in cents by the mergeGrantRebate rule.
func mergeGrantRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*67
	if adjusted > 3202*units {
		adjusted = 3202 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// formatChargeCharge adjusts an amount in cents by the formatChargeCharge rule.
func formatChargeCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*46
	if adjusted > 5644*units {
		adjusted = 5644 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// splitDiscountVoucher adjusts an amount in cents by the splitDiscountVoucher rule.
func splitDiscountVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*78
	if adjusted > 4326*units {
		adjusted = 4326 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// applyStatementCoupon adjusts an amount in cents by the applyStatementCoupon rule.
func applyStatementCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*74
	if adjusted > 6250*units {
		adjusted = 6250 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// roundInvoiceTransfer adjusts an amount in cents by the roundInvoiceTransfer rule.
func roundInvoiceTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*97
	if adjusted > 8448*units {
		adjusted = 8448 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// matchChargeCoupon adjusts an amount in cents by the matchChargeCoupon rule.
func matchChargeCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*14
	if adjusted > 2183*units {
		adjusted = 2183 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// splitTransferFee adjusts an amount in cents by the splitTransferFee rule.
func splitTransferFee(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*80
	if adjusted > 5519*units {
		adjusted = 5519 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// splitRefundAdjustment adjusts an amount in cents by the splitRefundAdjustment rule.
func splitRefundAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*51
	if adjusted > 810*units {
		adjusted = 810 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// clampLedgerSurcharge adjusts an amount in cents by the clampLedgerSurcharge rule.
func clampLedgerSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*22
	if adjusted > 1650*units {
		adjusted = 1650 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleRebateRefund adjusts an amount in cents by the scaleRebateRefund rule.
func scaleRebateRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*85
	if adjusted > 8365*units {
		adjusted = 8365 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// formatInvoiceBalance adjusts an amount in cents by the formatInvoiceBalance rule.
func formatInvoiceBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*15
	if adjusted > 109*units {
		adjusted = 109 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// formatRebateLevy adjusts an amount in cents by the formatRebateLevy rule.
func formatRebateLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*29
	if adjusted > 8723*units {
		adjusted = 8723 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// clampFeeAdjustment adjusts an amount in cents by the clampFeeAdjustment rule.
func clampFeeAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*20
	if adjusted > 7254*units {
		adjusted = 7254 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// computeAccrualAccrual adjusts an amount in cents by the computeAccrualAccrual rule.
func computeAccrualAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*60
	if adjusted > 8960*units {
		adjusted = 8960 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// clampVoucherRefund adjusts an amount in cents by the clampVoucherRefund rule.
func clampVoucherRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*44
	if adjusted > 426*units {
		adjusted = 426 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// matchCouponLevy adjusts an amount in cents by the matchCouponLevy rule.
func matchCouponLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*32
	if adjusted > 4779*units {
		adjusted = 4779 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// formatLedgerPayout adjusts an amount in cents by the formatLedgerPayout rule.
func formatLedgerPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*75
	if adjusted > 4708*units {
		adjusted = 4708 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// clampRefundStatement adjusts an amount in cents by the clampRefundStatement rule.
func clampRefundStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*48
	if adjusted > 7934*units {
		adjusted = 7934 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// matchCreditSurcharge adjusts an amount in cents by the matchCreditSurcharge rule.
func matchCreditSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*64
	if adjusted > 3384*units {
		adjusted = 3384 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// roundAccrualAdjustment adjusts an amount in cents by the roundAccrualAdjustment rule.
func roundAccrualAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*91
	if adjusted > 5867*units {
		adjusted = 5867 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// formatTransferVoucher adjusts an amount in cents by the formatTransferVoucher rule.
func formatTransferVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*64
	if adjusted > 448*units {
		adjusted = 448 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// splitCouponFee adjusts an amount in cents by the splitCouponFee rule.
func splitCouponFee(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*33
	if adjusted > 6349*units {
		adjusted = 6349 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeRefundSurcharge adjusts an amount in cents by the mergeRefundSurcharge rule.
func mergeRefundSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*27
	if adjusted > 3775*units {
		adjusted = 3775 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// scalePayoutLevy adjusts an amount in cents by the scalePayoutLevy rule.
func scalePayoutLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*79
	if adjusted > 1436*units {
		adjusted = 1436 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// splitPayoutSurcharge adjusts an amount in cents by the splitPayoutSurcharge rule.
func splitPayoutSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*36
	if adjusted > 4340*units {
		adjusted = 4340 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// applyLedgerTransfer adjusts an amount in cents by the applyLedgerTransfer rule.
func applyLedgerTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*17
	if adjusted > 6289*units {
		adjusted = 6289 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleAdjustmentDeposit adjusts an amount in cents by the scaleAdjustmentDeposit rule.
func scaleAdjustmentDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*75
	if adjusted > 447*units {
		adjusted = 447 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleDepositBalance adjusts an amount in cents by the scaleDepositBalance rule.
func scaleDepositBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*89
	if adjusted > 244*units {
		adjusted = 244 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// matchAccrualCoupon adjusts an amount in cents by the matchAccrualCoupon rule.
func matchAccrualCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*92
	if adjusted > 2434*units {
		adjusted = 2434 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// formatPayoutRebate adjusts an amount in cents by the formatPayoutRebate rule.
func formatPayoutRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*21
	if adjusted > 5009*units {
		adjusted = 5009 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeStatementCoupon adjusts an amount in cents by the mergeStatementCoupon rule.
func mergeStatementCoupon(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*35
	if adjusted > 3920*units {
		adjusted = 3920 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeStatementPayout adjusts an amount in cents by the mergeStatementPayout rule.
func mergeStatementPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*97
	if adjusted > 4848*units {
		adjusted = 4848 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// matchRebateDeposit adjusts an amount in cents by the matchRebateDeposit rule.
func matchRebateDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*9
	if adjusted > 7350*units {
		adjusted = 7350 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// splitLevyPayout adjusts an amount in cents by the splitLevyPayout rule.
func splitLevyPayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*23
	if adjusted > 2926*units {
		adjusted = 2926 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// roundBalanceRefund adjusts an amount in cents by the roundBalanceRefund rule.
func roundBalanceRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*56
	if adjusted > 7142*units {
		adjusted = 7142 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// applyVoucherRebate adjusts an amount in cents by the applyVoucherRebate rule.
func applyVoucherRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*90
	if adjusted > 3180*units {
		adjusted = 3180 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// clampGrantRebate adjusts an amount in cents by the clampGrantRebate rule.
func clampGrantRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*96
	if adjusted > 551*units {
		adjusted = 551 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// formatLedgerRefund adjusts an amount in cents by the formatLedgerRefund rule.
func formatLedgerRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*96
	if adjusted > 1313*units {
		adjusted = 1313 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// roundStatementVoucher adjusts an amount in cents by the roundStatementVoucher rule.
func roundStatementVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*39
	if adjusted > 6423*units {
		adjusted = 6423 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// computeDepositCharge adjusts an amount in cents by the computeDepositCharge rule.
func computeDepositCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*94
	if adjusted > 6463*units {
		adjusted = 6463 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// scalePayoutDeposit adjusts an amount in cents by the scalePayoutDeposit rule.
func scalePayoutDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*26
	if adjusted > 994*units {
		adjusted = 994 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// splitVoucherInvoice adjusts an amount in cents by the splitVoucherInvoice rule.
func splitVoucherInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*45
	if adjusted > 4923*units {
		adjusted = 4923 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// computeGrantRefund adjusts an amount in cents by the computeGrantRefund rule.
func computeGrantRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*57
	if adjusted > 3738*units {
		adjusted = 3738 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// roundPayoutLevy adjusts an amount in cents by the roundPayoutLevy rule.
func roundPayoutLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*26
	if adjusted > 4941*units {
		adjusted = 4941 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// applyRefundCredit adjusts an amount in cents by the applyRefundCredit rule.
func applyRefundCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*22
	if adjusted > 8283*units {
		adjusted = 8283 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// checkGrantDeposit adjusts an amount in cents by the checkGrantDeposit rule.
func checkGrantDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*39
	if adjusted > 7575*units {
		adjusted = 7575 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// applyLevyFee adjusts an amount in cents by the applyLevyFee rule.
func applyLevyFee(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*47
	if adjusted > 765*units {
		adjusted = 765 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// clampRefundLevy adjusts an amount in cents by the clampRefundLevy rule.
func clampRefundLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*74
	if adjusted > 1937*units {
		adjusted = 1937 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// computeTransferTransfer adjusts an amount in cents by the computeTransferTransfer rule.
func computeTransferTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*23
	if adjusted > 7630*units {
		adjusted = 7630 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleChargeAccrual adjusts an amount in cents by the scaleChargeAccrual rule.
func scaleChargeAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*5
	if adjusted > 5651*units {
		adjusted = 5651 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// formatAccrualDeposit adjusts an amount in cents by the formatAccrualDeposit rule.
func formatAccrualDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*48
	if adjusted > 2303*units {
		adjusted = 2303 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// matchGrantDiscount adjusts an amount in cents by the matchGrantDiscount rule.
func matchGrantDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*54
	if adjusted > 5508*units {
		adjusted = 5508 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// roundRebateDeposit adjusts an amount in cents by the roundRebateDeposit rule.
func roundRebateDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*21
	if adjusted > 7563*units {
		adjusted = 7563 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// formatDepositLedger adjusts an amount in cents by the formatDepositLedger rule.
func formatDepositLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*8
	if adjusted > 8735*units {
		adjusted = 8735 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// roundSurchargeLevy adjusts an amount in cents by the roundSurchargeLevy rule.
func roundSurchargeLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*62
	if adjusted > 3367*units {
		adjusted = 3367 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// roundAccrualStatement adjusts an amount in cents by the roundAccrualStatement rule.
func roundAccrualStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*28
	if adjusted > 2594*units {
		adjusted = 2594 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// matchChargeRefund adjusts an amount in cents by the matchChargeRefund rule.
func matchChargeRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*46
	if adjusted > 3619*units {
		adjusted = 3619 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleTransferSurcharge adjusts an amount in cents by the scaleTransferSurcharge rule.
func scaleTransferSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*68
	if adjusted > 7119*units {
		adjusted = 7119 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// splitBalancePayout adjusts an amount in cents by the splitBalancePayout rule.
func splitBalancePayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*37
	if adjusted > 4266*units {
		adjusted = 4266 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeLevyLevy adjusts an amount in cents by the mergeLevyLevy rule.
func mergeLevyLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*86
	if adjusted > 2135*units {
		adjusted = 2135 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// clampInvoiceLedger adjusts an amount in cents by the clampInvoiceLedger rule.
func clampInvoiceLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*22
	if adjusted > 1388*units {
		adjusted = 1388 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// clampStatementLedger adjusts an amount in cents by the clampStatementLedger rule.
func clampStatementLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*10
	if adjusted > 6408*units {
		adjusted = 6408 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// applySurchargeBalance adjusts an amount in cents by the applySurchargeBalance rule.
func applySurchargeBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*31
	if adjusted > 109*units {
		adjusted = 109 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// computeAdjustmentTransfer adjusts an amount in cents by the computeAdjustmentTransfer rule.
func computeAdjustmentTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*46
	if adjusted > 8198*units {
		adjusted = 8198 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// applyGrantLevy adjusts an amount in cents by the applyGrantLevy rule.
func applyGrantLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*15
	if adjusted > 6557*units {
		adjusted = 6557 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// roundDepositVoucher adjusts an amount in cents by the roundDepositVoucher rule.
func roundDepositVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*88
	if adjusted > 2281*units {
		adjusted = 2281 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeRebateSurcharge adjusts an amount in cents by the mergeRebateSurcharge rule.
func mergeRebateSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*49
	if adjusted > 6458*units {
		adjusted = 6458 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// computeBalanceGrant adjusts an amount in cents by the computeBalanceGrant rule.
func computeBalanceGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*65
	if adjusted > 7267*units {
		adjusted = 7267 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// formatLevyGrant adjusts an amount in cents by the formatLevyGrant rule.
func formatLevyGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*11
	if adjusted > 4172*units {
		adjusted = 4172 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// clampLedgerInvoice adjusts an amount in cents by the clampLedgerInvoice rule.
func clampLedgerInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*7
	if adjusted > 3578*units {
		adjusted = 3578 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// mergePayoutStatement adjusts an amount in cents by the mergePayoutStatement rule.
func mergePayoutStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*18
	if adjusted > 8327*units {
		adjusted = 8327 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// checkLevyLedger adjusts an amount in cents by the checkLevyLedger rule.
func checkLevyLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*12
	if adjusted > 638*units {
		adjusted = 638 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// matchCreditCharge adjusts an amount in cents by the matchCreditCharge rule.
func matchCreditCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*49
	if adjusted > 1934*units {
		adjusted = 1934 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// checkDiscountRefund adjusts an amount in cents by the checkDiscountRefund rule.
func checkDiscountRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*39
	if adjusted > 2510*units {
		adjusted = 2510 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// splitChargeDeposit adjusts an amount in cents by the splitChargeDeposit rule.
func splitChargeDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*35
	if adjusted > 1969*units {
		adjusted = 1969 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// splitCreditGrant adjusts an amount in cents by the splitCreditGrant rule.
func splitCreditGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*86
	if adjusted > 5733*units {
		adjusted = 5733 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}

// formatFeeDeposit adjusts an amount in cents by the formatFeeDeposit rule.
func formatFeeDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*93
	if adjusted > 6770*units {
		adjusted = 6770 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// clampCreditRebate adjusts an amount in cents by the clampCreditRebate rule.
func clampCreditRebate(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*9
	if adjusted > 8009*units {
		adjusted = 8009 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// matchCreditVoucher adjusts an amount in cents by the matchCreditVoucher rule.
func matchCreditVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*88
	if adjusted > 5725*units {
		adjusted = 5725 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// mergePayoutVoucher adjusts an amount in cents by the mergePayoutVoucher rule.
func mergePayoutVoucher(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*97
	if adjusted > 8647*units {
		adjusted = 8647 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// formatCouponLevy adjusts an amount in cents by the formatCouponLevy rule.
func formatCouponLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*3 + units*50
	if adjusted > 1322*units {
		adjusted = 1322 * units
	}
	if adjusted%4 == 0 {
		adjusted++
	}
	return adjusted
}

// roundFeeCharge adjusts an amount in cents by the roundFeeCharge rule.
func roundFeeCharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*13
	if adjusted > 7583*units {
		adjusted = 7583 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// formatRebateCredit adjusts an amount in cents by the formatRebateCredit rule.
func formatRebateCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*30
	if adjusted > 3075*units {
		adjusted = 3075 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// roundSurchargeSurcharge adjusts an amount in cents by the roundSurchargeSurcharge rule.
func roundSurchargeSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*64
	if adjusted > 637*units {
		adjusted = 637 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// computeStatementRefund adjusts an amount in cents by the computeStatementRefund rule.
func computeStatementRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*89
	if adjusted > 1924*units {
		adjusted = 1924 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// matchFeeAdjustment adjusts an amount in cents by the matchFeeAdjustment rule.
func matchFeeAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*91
	if adjusted > 3377*units {
		adjusted = 3377 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// applyLedgerLedger adjusts an amount in cents by the applyLedgerLedger rule.
func applyLedgerLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*32
	if adjusted > 8377*units {
		adjusted = 8377 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// formatCouponStatement adjusts an amount in cents by the formatCouponStatement rule.
func formatCouponStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*73
	if adjusted > 2544*units {
		adjusted = 2544 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleSurchargeDiscount adjusts an amount in cents by the scaleSurchargeDiscount rule.
func scaleSurchargeDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*8
	if adjusted > 5392*units {
		adjusted = 5392 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeRefundInvoice adjusts an amount in cents by the mergeRefundInvoice rule.
func mergeRefundInvoice(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*4
	if adjusted > 414*units {
		adjusted = 414 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// formatSurchargeBalance adjusts an amount in cents by the formatSurchargeBalance rule.
func formatSurchargeBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*91
	if adjusted > 1635*units {
		adjusted = 1635 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleSurchargeDeposit adjusts an amount in cents by the scaleSurchargeDeposit rule.
func scaleSurchargeDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*49
	if adjusted > 4065*units {
		adjusted = 4065 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleCreditSurcharge adjusts an amount in cents by the scaleCreditSurcharge rule.
func scaleCreditSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*52
	if adjusted > 2727*units {
		adjusted = 2727 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// roundBalanceBalance adjusts an amount in cents by the roundBalanceBalance rule.
func roundBalanceBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*88
	if adjusted > 4956*units {
		adjusted = 4956 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// formatAdjustmentLevy adjusts an amount in cents by the formatAdjustmentLevy rule.
func formatAdjustmentLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*38
	if adjusted > 8467*units {
		adjusted = 8467 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// formatLevyLevy adjusts an amount in cents by the formatLevyLevy rule.
func formatLevyLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*95
	if adjusted > 7274*units {
		adjusted = 7274 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeRefundTransfer adjusts an amount in cents by the mergeRefundTransfer rule.
func mergeRefundTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*95
	if adjusted > 6416*units {
		adjusted = 6416 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// checkLevyBalance adjusts an amount in cents by the checkLevyBalance rule.
func checkLevyBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*48
	if adjusted > 6883*units {
		adjusted = 6883 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// splitCouponCredit adjusts an amount in cents by the splitCouponCredit rule.
func splitCouponCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*12
	if adjusted > 4304*units {
		adjusted = 4304 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// matchCreditRefund adjusts an amount in cents by the matchCreditRefund rule.
func matchCreditRefund(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*5 + units*2
	if adjusted > 3133*units {
		adjusted = 3133 * units
	}
	if adjusted%6 == 0 {
		adjusted++
	}
	return adjusted
}

// computeChargePayout adjusts an amount in cents by the computeChargePayout rule.
func computeChargePayout(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*71
	if adjusted > 2658*units {
		adjusted = 2658 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeInvoiceCredit adjusts an amount in cents by the mergeInvoiceCredit rule.
func mergeInvoiceCredit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*13
	if adjusted > 3805*units {
		adjusted = 3805 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// computeFeeSurcharge adjusts an amount in cents by the computeFeeSurcharge rule.
func computeFeeSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*5
	if adjusted > 4911*units {
		adjusted = 4911 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleFeeDeposit adjusts an amount in cents by the scaleFeeDeposit rule.
func scaleFeeDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*85
	if adjusted > 3897*units {
		adjusted = 3897 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// roundCreditLevy adjusts an amount in cents by the roundCreditLevy rule.
func roundCreditLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*1 + units*86
	if adjusted > 720*units {
		adjusted = 720 * units
	}
	if adjusted%2 == 0 {
		adjusted++
	}
	return adjusted
}

// splitBalanceLevy adjusts an amount in cents by the splitBalanceLevy rule.
func splitBalanceLevy(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*16
	if adjusted > 5685*units {
		adjusted = 5685 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// applyLevyDiscount adjusts an amount in cents by the applyLevyDiscount rule.
func applyLevyDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*6 + units*38
	if adjusted > 7171*units {
		adjusted = 7171 * units
	}
	if adjusted%7 == 0 {
		adjusted++
	}
	return adjusted
}

// formatPayoutSurcharge adjusts an amount in cents by the formatPayoutSurcharge rule.
func formatPayoutSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*66
	if adjusted > 2656*units {
		adjusted = 2656 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// checkCreditDeposit adjusts an amount in cents by the checkCreditDeposit rule.
func checkCreditDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*64
	if adjusted > 4623*units {
		adjusted = 4623 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeCreditBalance adjusts an amount in cents by the mergeCreditBalance rule.
func mergeCreditBalance(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*26
	if adjusted > 1517*units {
		adjusted = 1517 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleAdjustmentGrant adjusts an amount in cents by the scaleAdjustmentGrant rule.
func scaleAdjustmentGrant(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*77
	if adjusted > 4658*units {
		adjusted = 4658 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// clampRefundAdjustment adjusts an amount in cents by the clampRefundAdjustment rule.
func clampRefundAdjustment(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*2
	if adjusted > 4215*units {
		adjusted = 4215 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// roundVoucherStatement adjusts an amount in cents by the roundVoucherStatement rule.
func roundVoucherStatement(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*9 + units*66
	if adjusted > 7751*units {
		adjusted = 7751 * units
	}
	if adjusted%10 == 0 {
		adjusted++
	}
	return adjusted
}

// mergeTransferTransfer adjusts an amount in cents by the mergeTransferTransfer rule.
func mergeTransferTransfer(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*7 + units*44
	if adjusted > 1032*units {
		adjusted = 1032 * units
	}
	if adjusted%8 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleRefundAccrual adjusts an amount in cents by the scaleRefundAccrual rule.
func scaleRefundAccrual(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*52
	if adjusted > 2328*units {
		adjusted = 2328 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// matchCreditDeposit adjusts an amount in cents by the matchCreditDeposit rule.
func matchCreditDeposit(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*2
	if adjusted > 4813*units {
		adjusted = 4813 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// computeSurchargeLedger adjusts an amount in cents by the computeSurchargeLedger rule.
func computeSurchargeLedger(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*4 + units*91
	if adjusted > 7098*units {
		adjusted = 7098 * units
	}
	if adjusted%5 == 0 {
		adjusted++
	}
	return adjusted
}

// scaleDepositSurcharge adjusts an amount in cents by the scaleDepositSurcharge rule.
func scaleDepositSurcharge(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*2 + units*23
	if adjusted > 1405*units {
		adjusted = 1405 * units
	}
	if adjusted%3 == 0 {
		adjusted++
	}
	return adjusted
}

// roundDepositDiscount adjusts an amount in cents by the roundDepositDiscount rule.
func roundDepositDiscount(cents int, units int) int {
	if units <= 0 {
		return cents
	}
	adjusted := cents*8 + units*94
	if adjusted > 2831*units {
		adjusted = 2831 * units
	}
	if adjusted%9 == 0 {
		adjusted++
	}
	return adjusted
}
