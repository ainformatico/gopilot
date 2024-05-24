# gopilot
Enjoy Github Copilot on the CLI. Especially powerful when combined with `tmux`.

## nvim
![nvim](https://github.com/ainformatico/gopilot/assets/161387/2b087c15-c9ed-477c-95b5-a5d2bbbfa9c8)

## cli
![cli](https://github.com/ainformatico/gopilot/assets/161387/920f88db-71e0-4fc6-84fd-2795ecf87d23)

## Disclaimer
This project go as-if. It will be my WIP project to learn Go. You have been warned.

## Why?
I am a heavy (n)vim user. I have been using it for 14 years now.
When Copilot Chat was out I decided to try VSCode with the Chat integration. I have to admit that it was a great experience.
Well, the Chat was, the editor wasn't.
I spent *many days* trying to configure VSCode to behave like my nvim setup. They are completely different, so there's no point trying to make them behave the same way.
Still, I wanted to use Copilot Chat from my editor!
So, as any good developer would do, I decided to write my own client for it.
This coincided with me wanting to learn Go, so here we are.

## Why not a plugin for nvim?
I am also a `tmux` heavy user, and I wanted to have the chat available on any terminal, not just on nvim.
Sometimes I ask questions about certain CLI commands and not just about code.

If you are looking for a great plugin to integrate Copilot Chat in your nvim, take a look at [CopilotChat.nvim](https://github.com/CopilotC-Nvim/CopilotChat.nvim)

## Technical considerations
### Authentication token
This project assumes that you have `~/.config/github-copilot/hosts.json` that contains your Github Token for Copilot.
Note that this is a UNIX path. I am not interested in adding Windows support at the moment. Feel free to open a PR or fork if you want to add it.

The fastest way to get your token is:
1. Install VSCode
2. Install the Copilot extension
3. The extension should ask you for your Github details and create the token file
4. Done!

### Request headers
I ported (using Copilot) the great work from [CopilotChat.nvim](https://github.com/CopilotC-Nvim/CopilotChat.nvim), so I am using the same request headers as they are. Most of the values are randomized, but it is up to you to check and use the desired values.

## Debugging
If you are having issues or are developing this project, you can run:

```bash
gopilot -d
```
This will create a `gopilot.log` file in the project directory with the debug information.

## How to use gopilot
1. Clone this repo
2. Run `go get`
3. Run `make build`
4. Run `make install`
5. Run `gopilot`
6. Enjoy!

## Chat
### Keybindings
* `Ctrl + j`: Sends the message
* `Ctrl + l`: Clears the chat and restarts the session
* `Ctrl + c`: Quit
* `enter`: Allows for multi-line messages
* `Ctrl + p`, `PageUp`: Scroll up in the chat viewport
* `Ctrl + n`, `PageDown`: Scroll down in the chat viewport
* `Ctrl + r`: Used only for debugging. Reloads the Github token
