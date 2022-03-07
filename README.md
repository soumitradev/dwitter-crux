# Dwitter Go API

I'm trying to make a Twitter clone using Go, GraphQL, and Prisma with a Postgresql backend.

I use `make` to make my workflow easier (no pun intended), the commands are:

`make run` : Run only the API. Don't migrate databases.

`make migrate` : Run through only migration of the database. If DB cannot be migrated, you will need to delete it.

`make clean` : Run through only the deletion of the database. This will delete the database.

`make kill` : Kill any processes that are currently accessing the database. This is only to be used if an error is raised when running `make clean`

The prisma schema is [here](./prisma/schema.prisma)

**NOTE:** You need `ffmpeg` added to your path to run this. It uses ffmpeg to generate thumbnails for videos uploaded.

> .env contains ACCESS_SECRET, REFRESH_SECRET, DISCORD_CLIENT_ID, DISCORD_CLIENT_SECRET, SENDGRID_API_KEY, and SENDGRID_SENDER_EMAIL_ADDR

> cdn_key.json is the key to Google Firebase

**TODO:**

- Add more OAuth providers
- If username/email is already taken, allow them to set a new one
- If user has already signed up, log them in when using OAuth flow
- Advanced Search for Dweets
- Infinite recursion: https://pkg.go.dev/github.com/graphql-go/graphql#Field I'm starting to think this is possible, and I'll have to rewrite half of my backend code if I manage to do it, but maybe that's just me going insane as I work on this project more.
- 10000000x better decision logic in db_externals.go, where I collapse parameters into a single variable and make decisions based on info I extract from that single variable, kind of like an opcode.
- Frontend in Vue

**TODO: (Advanced)**

- Add support for becoming an OAuth identity provider

The entire API is now ready. I haven't tested for every scenario, because it would take too much time to do it, so I'll fix any bugs I find in the future when working on the frontend.

I'm very happy with how this turned out. I learned a **lot**.

This is by far my biggest project ever. Yes. Including BruhOS, including the chip8 hardware emulation project I did, even the fullstack Node project I made last year.

This was really fucking fun, especially working with my homie [@PseudoCodeNerd / @Madhav Sharma / @mdvsh](https://github.com/mdvsh).

His support and enthusiasm kept me going all through this. He helped me debug some really tough logic issues too. I wouldn't have made any of this without his help and inspiration.

I literally can't imagine myself building the frontend without him. He's really helpful and knowledgeable.

This (hopefully) will be integrated into a bigger Dwitter project later.

## Update

So um... College/Uni happened. I am pretty busy now, but I really want to complete this project some other time.

Everything is done, I just need to wire the frontend to the backend now. Kinda excited. This project was both painful and fun.
