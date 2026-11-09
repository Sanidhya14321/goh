class TimeoutError_(Exception):
    pass


class PaymentError(Exception):
    pass


def charge_with_retry(gateway, invoice_id, amount, attempts=3):
    """Charge once. A timeout may mean the charge went through, so before each
    retry the gateway is asked whether invoice_id was already charged; a
    charge is never sent twice. Other errors are raised immediately."""
    for attempt in range(attempts):
        try:
            return gateway.charge(invoice_id, amount)
        except TimeoutError_:
            existing = gateway.lookup(invoice_id)
            if existing is not None:
                return existing
            if attempt == attempts - 1:
                raise PaymentError("payment timed out")
