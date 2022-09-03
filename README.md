# GithubWebhookProxy

A proxy to filter out star spam. 
After a user stars a repo, any subsequent stars of the same repo within the next 15 minutes will be ignored.
I have only tested this with discord webhooks, so it may not work correctly with other sites.

## How it works

Visit the homepage and paste your webhook url in the input.
Use the url the site gives you as webhook url. That's it!
When the webhook is fired and the event is a star event, the ID of the repo and the username of the user are stored.
If they star again within the next 15 minutes, the request is dropped.
