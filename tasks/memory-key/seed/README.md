# inbox

Notes in `inbox/` are encrypted with the project's archive word. The format is the
default of this command with the archive word as the password:

    openssl enc -aes-256-cbc -pbkdf2 -iter 300000 -md sha256 -a -in inbox/note.enc -d -pass pass:<archive word>

A wrong word fails with "bad decrypt".
