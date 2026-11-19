# recs

Billing records arrive as `.rec` files. `data/samples/` holds records known to be valid;
`data/inbox/` holds newly received ones. Package `recfmt` parses the layout below.

## The .rec format

Each file is one record: a 12-byte header followed by the payload.

| offset | size | field                                                  |
|--------|------|--------------------------------------------------------|
| 0      | 4    | magic, the ASCII bytes `RECF`                          |
| 4      | 1    | format version, always 2                               |
| 5      | 1    | kind: 1 = order, 2 = refund, 3 = adjustment            |
| 6      | 2    | payload length in bytes, unsigned, big-endian          |
| 8      | 4    | check value                                            |
| 12     | n    | payload                                                |

The payload is one `key=value` line per field, in field order, each line ending in `\n`
(including the last). Keys are non-empty and contain neither `=` nor `\n`; values contain no
`\n`. A record may have no fields (empty payload).

The check value is derived from a 32-bit CRC by the producer of the files; exactly how is not
documented. A reader must reject a record whose check value does not match.
