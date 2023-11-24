package libXray

type DialerController interface {
	ProtectFd(int) bool
}
