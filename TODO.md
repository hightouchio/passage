- Encrypt private keys.
- Use AWS Secret store
- Add status
- Encrypt SSH keys in DB
- Status stats + monitors
- [https://aws.amazon.com/premiumsupport/knowledge-center/custom-private-primary-address-ec2/](https://aws.amazon.com/premiumsupport/knowledge-center/custom-private-primary-address-ec2/)

## DONE
- Re-enable authentication checking for reverse tunnels
- Re-enable authentication for normal tunnels
- Dynamic user (defaults to hightouch still)
- Better logging
- Try to find a way to make the client blind to whether its a normal or reverse proxy. It shouldn't matter.
- Datadog integration
- Add CI
- Upgrade Go 1.16
  - Use [signal.Notify](https://millhouse.dev/posts/graceful-shutdowns-in-golang-with-signal-notify-context) 
- Should detect when a tunnel is removed
- Add tester
