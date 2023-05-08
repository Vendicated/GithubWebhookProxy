# GithubWebhookProxy

A proxy to filter out star spam. 
After a user stars a repo, any subsequent stars of the same repo within the next 15 minutes will be ignored.
I have only tested this with discord webhooks, so it may not work correctly with other sites.

## How it works

Visit the homepage and paste your webhook url in the input.
Use the url the site gives you as webhook url. That's it!
When the webhook is fired and the event is a star event, the ID of the repo and the username of the user are stored.
If they star again within the next 15 minutes, the request is dropped.

## Selfhosting

Building ghwp is trivial:

```sh
git clone https://github.com/Vendicated/GithubWebhookProxy
cd GithubWebhookProxy
go build
```

This outputs a binary `ghwp` (or `ghwp.exe` on Windows) that when run will start the server on port 1337

You should then use a web server like Caddy (recommended! [Example Caddyfile that runs behind Cloudflare proxy](/Caddyfile)), NGINX, or Apache to reverse proxy it.
Make sure that the X-Forwarded-For header is set to the requester's ip and has not been spoofed with, as it is used to verify that webhook post requests are genuine requests coming from GitHub
