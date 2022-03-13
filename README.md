# Dwitter Go API

## What this is?

I'm trying to make a Twitter clone using Go, GraphQL, and Prisma with a Postgresql backend, and a frontend built with Tailwind CSS, Vue, and apollo.

This fork uses session authentication, stores sessions in Redis (port 6420), caches API results (port 6421) on a redis LRU cache, and has an option to subscribe to email notifications for a dweet or user.

The prisma schema is [here](./schema.prisma). This is the single document that will explain to you end to end the way my API functions and stores data.

The redis session store has been configured according to [https://redis.io/topics/quickstart](https://redis.io/topics/quickstart), but at port 6420 and 6421. The config and start script files are in the root of this repo.

## How do I run it?

I use `make` to make my workflow easier (no pun intended), the commands are:

`make run` : Run the API, and build the frontend. Not recommended unless _really_ needed. Very bad choice for development, use `make api` or `make serve` instead based on your task.

`make api` : Run the API only

`make migrate` : Run through only migration of the database. If DB cannot be migrated, you will need to delete it.

`make clean` : Run through only the deletion of the database. This will delete the database.

`make psql` : Start the PSQL server

`make redis` : Start the redis servers

`make kill` : Kill any processes that are currently accessing the PSQL database. This is only to be used if an error is raised when running `make clean`

`make api` : Build the frontend

`make serve` : Serve only the frontend with hot-reload, auto-refresh, and everything

**NOTE:** You need `ffmpeg` added to your path to run this. It uses ffmpeg to generate thumbnails for videos uploaded.

> .env contains ACCESS_SECRET, REFRESH_SECRET, DISCORD_CLIENT_ID, DISCORD_CLIENT_SECRET, SENDGRID_API_KEY, SENDGRID_SENDER_EMAIL_ADDR, REDIS_6420_PASS, and REDIS_6421_PASS

> cdn_key.json is the key to Google Firebase

## Why?

This is a part of a uni club induction task.

The tasks assigned to me were:

> 1. Allow users to add a profile picture when registering. They should be able to upload an image as a part of the request. This image must then be stored on a cloud storage service and must be served from a CDN. (Firebase, Cloudinary, etc.)
> 2. Add email verification upon user registration (Only for users registered via email/password).
> 3. Migrate to session-based authentication and store user sessions in a data store like Redis
> 4. Configure your Go Api to use Redis to cache page results. Use LRU for cache updates, and set an expiry time for the cached results.
> 5. Brownie Points: Add a mailing list. Users can choose to receive email notifications for a particular tweet based on tags/user.
> 6. Extra Brownie Points: Add functionality to schedule your tweets. Replace scheduling features provided by email service providers with your implementation using a Message Broker and a Task Queue.

I had 11 days for this task, of which I could utilize only about 7-8 days (I was travelling, and I had to spend time on getting stuff ready to move to campus)

I have finished the first 5/6 of these tasks, the last one I speedran in the last 20 minutes.

[This document](./Caching.md) outlines some of my thoughts on how I implemented the LRU cache on redis.

## Other notes

This is by far my biggest project ever. Yes. Including BruhOS, including the chip8 hardware emulation project I did, even the fullstack Node project I made last year.

This was really fun, especially working with my homie [@PseudoCodeNerd / @Madhav Sharma / @mdvsh](https://github.com/mdvsh).

His support and enthusiasm kept me going all through this. He helped me debug some really tough logic issues too. I wouldn't have made any of this without his help and inspiration.

I literally can't imagine myself building the frontend without him. He's really helpful and knowledgeable.

This (hopefully) will be integrated into a bigger Dwitter project later.
