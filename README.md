# ipcoin

ipcoin is the world's first recentralized cryptocurrency.

Project web links:

* Website: https://ipcoin.hypoxia.dev/
* HackerNews: https://news.ycombinator.com/item?id=44858431

This is a cryptocurrency parody project. See the project website for more information.

## How does it work?

ipcoin is a simple web app. Every hour, every IP address in existance gets 1 ipcoin. You can transfer ipcoins between IP
addresses and publish comments. All transfers and comments are public and published alongside the IP address that made
them.

Comments are automatically moderated once per minute
using [OpenAI's free moderation API](https://platform.openai.com/docs/guides/moderation).

The frontend is built using TypeScript, React Router 7, and Tailwind CSS.

The backend is built using Golang, gRPC Gateway, and PostgreSQL.

The ipcoin balance for a given IP address is calculated using the hard-coded start time, the transfer history, and the
current time.

TODO Write a more detailed explanation some other day. I have to go get ice cream now.

## Where's the frontend code?

I used [Tailwind Plus](https://tailwindcss.com/plus#pricing) and [FontAwesome Pro](https://fontawesome.com/plans) to
build the frontend, which are paid products that are not open source. To open source the frontend, I'd need to re-read
the licenses and maybe write instructions on how to use the code that doesn't make it into VCS. I haven't done that yet,
but maybe I'll find a way to properly open source the frontend code in the future.
