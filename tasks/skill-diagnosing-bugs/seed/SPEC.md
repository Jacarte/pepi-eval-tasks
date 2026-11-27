# checkout specification

`checkout` is a small module of Go 1.22 code, standard library only. It quotes the
total of a cart after promotion codes.

| Package           | Purpose                                                     |
|-------------------|-------------------------------------------------------------|
| `checkout/pricing`| applies promotions to a subtotal, in cents                  |
| `checkout/promo`  | resolves the codes a customer entered against the book      |
| `checkout`        | `Quote`: cart in, total in cents out                        |

## Promotions

A promotion is either a percentage (`Percent`, 1 to 100) or a fixed amount
(`OffCents`, more than 0). Promotions are applied in sequence, to the running
total:

- A percentage takes `total * Percent / 100` off, rounded down to a whole cent.
- A fixed amount takes `OffCents` off, and the total never goes below zero.

Codes combine in the order the customer entered them. The same two codes in a
different order can give a different total; that is intended.

## Codes

- `promo.Resolve(codes, book)` returns one promotion per distinct code, in the
  order the codes were first entered. A code entered twice counts once, at its
  first position.
- A code that is not in the book is an error: `unknown promotion code "<code>"`.

## Quote

- `checkout.Quote(cart, book)` returns `pricing.Apply(cart.SubtotalCents,
  promotions)` for the promotions `promo.Resolve(cart.Codes, book)` gives.
- A cart with no codes is quoted at its subtotal.
- The result of `Quote` depends only on the cart and the book. Quoting the same
  cart again gives the same total.
