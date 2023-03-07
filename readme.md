## Todo

- use `const` in main.go
- Add a button to recognize and open URL in a new tab
- Add a program that listens SSE and automatically copies the content to clipboard (pbcopy xsel xclip wl-clipboard)
  - iOS doesn't always runs Tailscale because of battery consumption, so don't think about incl. the program there. We also don't have control over when iOS can suspend the apps
  - Add a checkbox to automatically copy upon receiving SSE
    - Always send SSE on client at `/`, the website should give an option to automatically copy or not
      - Use navigator.isonline and direct pings to device domain (:80) to restore SSE connection (reconnect)
      - Hide the option on small screens? Or identify iOS Safari is running to tell users why it won't be reliable
      - sending mutiple text is actually painful
- mention that back tap was used to be called tap back and used to be situated somewhere bottom down in the list with the help of a youtube video
- Add startup.fish to dotfiles
- tell wait might be region dependent and not reliable https://www.reddit.com/r/shortcuts/comments/kvcg39/setting_a_delay_shorter_than_1_second/gixhks5/
- mention the fact that explicit copy paste is better, to avoid unknowningly copy sensitive data
- other options, sound recog., tap back, and assisstive are not as most covenient and least distracting
- SSL in short domain doesn't work. sd:1111 won't work
- content same ho xsel ka to na copy karo in telltail-sync
  - what about copy loop? received from SSE and initiated a copy will result in sending the result back to API as clip-notify will be invoked (check if this actually happens)
    - if it does, either use a flag to skip sending to API, or instead of a switch-match channel, use receive/send chan in sequence (nah, ye kaam nahi karega, check for sync/atomic/mutex lock or something, or maybe a simple flag will do)
- Vercel Sans, Public Sans, Inter, Lato, see last fontshare link in Firefox, Work Sans, Manrope, Source Sans Pro
- Mention the existence of clipman for wayland even if you have clipnotify and xsel installed

## To use this

1. Generate certs using tailscale for the computer you'd run this on
1. Properly locate those certs in the program (main.go)
1. Configure it on startup https://github.com/ajitid/dotfiles/blob/main/scripts/scripts/startup.fish
