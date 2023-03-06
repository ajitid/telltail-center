## Todo

- Add a button to recognize and open URL in a new tab
- Add a program that listens SSE and automatically copies the content to clipboard (pbcopy xsel xclip wl-clipboard)
  - iOS doesn't always runs Tailscale because of battery consumption, so don't think about incl. the program there. We also don't have control over when iOS can suspend the apps
  - Add a checkbox to automatically copy upon receiving SSE
    - Always send SSE on client at `/`, the website should give an option to automatically copy or not
      - Use navigator.isonline and direct pings to device domain (:80) to restore SSE connection (reconnect)
      - Hide the option on small screens? Or identify iOS Safari is running to tell users why it won't be reliable
      - sending mutiple text is actually painful
- Add startup.fish to dotfiles

## To use this

1. Generate certs using tailscale for the computer you'd run this on
1. Properly locate those certs in the program (main.go)
1. Configure it on startup https://github.com/ajitid/dotfiles/blob/main/scripts/scripts/startup.fish
