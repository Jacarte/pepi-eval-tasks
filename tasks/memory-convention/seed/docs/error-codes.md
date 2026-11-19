# Error codes

Codes returned in the `error.code` field of API error responses.

| Code | HTTP | Meaning |
| --- | --- | --- |
| ERR-PAY-0001 | 500 | Internal error |
| ERR-PAY-0002 | 400 | Request body is not valid JSON |
| ERR-PAY-0003 | 400 | A required field is missing or empty |
| ERR-PAY-0004 | 404 | Payment not found |
| ERR-PAY-0005 | 404 | Customer not found |
| ERR-PAY-0006 | 409 | Payment is not in the authorized state |
| ERR-PAY-0007 | 400 | Currency is not supported |
| ERR-PAY-0008 | 400 | Amount must be a positive number of cents |
| ERR-PAY-0009 | 409 | Customer email is already registered |
