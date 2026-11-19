# inbox

Notes in `inbox/` are encrypted with the project's archive word as the password.
Decrypt one with:

    openssl enc -aes-256-cbc -pbkdf2 -iter 300000 -md sha256 -a -in inbox/note.enc -d -pass pass:<archive word>

A wrong word gives an error or unreadable output.
