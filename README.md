# Overview

Run this CLI application to easily handle the emails in your IMAP inbox (e.g. outlook365, gmail, etc.)

I created the app because my email client, while great and smart, is not good at filtering emails.

This application does not even have command-line arguments yet, and the only way to change its behavior is through its configuration files. Obviously this needs to change down the road.

To be fair: editing the configuration files is _very_ easy.

WARNING!

Again, at this point, the application works fairly well but it's only a MVP. It will not devour your emails, but it is brittle.

# Getting started

- Copy `secret.toml.template` to `secret.toml`
- Store your login credentials in `secret.toml` (you may need to create application-specific passwords... oauth not yet supported)
- Copy `config.toml.template` to `config.toml`
- Create all the rules you need
- Run `mailbiter`

# Concepts

So, what do we find in `config.toml`?

## Profiles

A profile represents an email account. It has simple settings, such as number of emails to retrieve, and the name of the secret containing your credentials in `secret.toml`

## Rules

A rule can either be a row rule ('rowrule') or a set rule ('setrule')

You are likely going to use **row rules** as they apply, individually, to each email retrieved from your inbox.

**Set rules** are not implemented!

When/if they are, here is what they will do: they will be run against filters applied to multiple emails. This will allow you, for instance, to say "I do not want to keep more than 3 emails from that sender."

**A rule** consists of a complex condition and _one or more_ actions.

## Conditions

Conditions are expressed using a syntax close to natural language. For instance:
```
subject contains 'you won' and sender contains 'lottery'
```

This basic syntax can be very powerful. Here are a few more examples:

- `subject contains 'reminder' and not (subject contains 'tomorrow')`
- `subject contains 'blue' or subject contains 'green'`
- `sender endswith '@example.com'`
- `subject in ['cat', 'dog', 'rabbit']`
- `now - date > duration('24h') and sender != 'chris@example.com'`
- `date < calendar('2020-02-03') and date > calendar('2020-02-01')`
- `'Seen' in flags`

## Actions

- `move to '<folder name>'`
- `copy to '<folder name>'`
- `delete`
- `info`
- `inspect`
- `run <path>` # TODO

# TODO

## Urgent

- Proper error handling
- Filter more than just inbox
- Implement command-line arguments

## Open points

Should we also provide email size as a criterion?

Sanity checks could be quickly performed before a run:
- ensure that all actionnames reference actual action entries (done)
- ensure that all accounts names reference actual account entries