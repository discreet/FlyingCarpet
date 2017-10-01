package main

import (
	"crypto/md5"
	"fmt"
	"github.com/dontpanic92/wxGo/wx"
	"math/rand"
	"net"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"time"
)

const DIAL_TIMEOUT = 60
const JOIN_ADHOC_TIMEOUT = 60
const FIND_MAC_TIMEOUT = 60

const OUTPUT_BOX_UPDATE = wx.ID_HIGHEST + 1
const PROGRESS_BAR_UPDATE = wx.ID_HIGHEST + 2
const PROGRESS_BAR_SHOW = wx.ID_HIGHEST + 3
const START_BUTTON_ENABLE = wx.ID_HIGHEST + 4

// need different thread event for each post to output box, so need helper function that's receiver on Transfer?

func main() {
	wx1 := wx.NewApp()
	mf := newGui()
	mf.Show()
	wx1.MainLoop()
	return
}

func (t *Transfer) mainRoutine(mode string) {
	receiveChan := make(chan bool)
	sendChan := make(chan bool)
	var n Network

	if mode == "send" {
		pwBytes := md5.Sum([]byte(t.Passphrase))
		prefix := pwBytes[:3]
		t.SSID = fmt.Sprintf("flyingCarpet_%x", prefix)

		if runtime.GOOS == "windows" {
			w := WindowsNetwork{Mode: "sending"}
			w.PreviousSSID = w.getCurrentWifi()
			n = w
			t.output("Got current wifi")
		} else if runtime.GOOS == "darwin" {
			n = MacNetwork{Mode: "sending"}
		}
		if !n.connectToPeer(t) {
			t.enableStartButton()
			t.output("Exiting mainRoutine.")
			return
		}

		if connected := t.sendFile(sendChan, n); connected == false {
			t.enableStartButton()
			t.output("Could not establish TCP connection with peer. Exiting mainRoutine.")
			return
		}
		t.output("Connected")
		sendSuccess := <-sendChan
		if !sendSuccess {
			t.enableStartButton()
			t.output("Exiting mainRoutine.")
			return
		}
		t.enableStartButton()
		t.output("Send complete, resetting WiFi and exiting.")

	} else if mode == "receive" {
		t.Passphrase = generatePassword()
		pwBytes := md5.Sum([]byte(t.Passphrase))
		prefix := pwBytes[:3]
		t.SSID = fmt.Sprintf("flyingCarpet_%x", prefix)
		t.output(fmt.Sprintf("=============================\n"+
			"Transfer password: %s\nPlease use this password on sending end when prompted to start transfer.\n"+
			"=============================\n", t.Passphrase))
		

		if runtime.GOOS == "windows" {
			n = WindowsNetwork{Mode: "receiving"}
		} else if runtime.GOOS == "darwin" {
			n = MacNetwork{Mode: "receiving"}
		}
		if !n.connectToPeer(t) {
			t.enableStartButton()
			t.output("Exiting mainRoutine.")
			return
		}

		go t.receiveFile(receiveChan, n)
		// wait for listener to be up
		listenerIsUp := <-receiveChan
		if !listenerIsUp {
			t.enableStartButton()
			t.output("Exiting mainRoutine.")
			return
		}
		// wait for reception to finish
		receiveSuccess := <-receiveChan
		if !receiveSuccess {
			t.enableStartButton()
			t.output("Exiting mainRoutine.")
			return
		}
		t.output("Reception complete, resetting WiFi and exiting.")
	}
	n.resetWifi(t)
}

func (t *Transfer) receiveFile(receiveChan chan bool, n Network) {
	
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(t.Port))
	if err != nil {
		n.teardown(t)
		t.output(fmt.Sprintf("Could not listen on :%d", t.Port))
		receiveChan <- false
		return
	}
	t.output("Listening on :" + strconv.Itoa(t.Port))
	receiveChan <- true
	for {
		conn, err := ln.Accept()
		if err != nil {
			n.teardown(t)
			t.output(fmt.Sprintf("Error accepting connection on :%d", t.Port))
			receiveChan <- false
			return
		}
		t.Conn = conn
		t.output("Connection accepted")
		go t.receiveAndAssemble(receiveChan, n)
	}
}

func (t *Transfer) sendFile(sendChan chan bool, n Network) bool {
	
	var conn net.Conn
	var err error
	for i := 0; i < DIAL_TIMEOUT; i++ {
		err = nil
		conn, err = net.DialTimeout("tcp", t.RecipientIP+":"+strconv.Itoa(t.Port), time.Millisecond*10)
		if err != nil {
			t.output(fmt.Sprintf("Failed connection %2d to %s, retrying.", i, t.RecipientIP))
			time.Sleep(time.Second * 1)
			continue
		}
		t.output("Successfully dialed peer.")
		t.Conn = conn
		go t.chunkAndSend(sendChan, n)
		return true
	}
	t.output(fmt.Sprintf("Waited %d seconds, no connection.", DIAL_TIMEOUT))
	return false
}

func generatePassword() string {
	const chars = "0123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"
	rand.Seed(time.Now().UTC().UnixNano())
	pwBytes := make([]byte, 8)
	for i := range pwBytes {
		pwBytes[i] = chars[rand.Intn(len(chars))]
	}
	return string(pwBytes)
}

func newGui() *MainFrame {

	mf := &MainFrame{}
	mf.Frame = wx.NewFrame(wx.NullWindow, wx.ID_ANY, "Flying Carpet")

	mf.SetSize(400, 400)

	// radio buttons box
	radioSizer := wx.NewBoxSizer(wx.HORIZONTAL)

	// peer os box
	peerSizer := wx.NewBoxSizer(wx.VERTICAL)
	radiobox1 := wx.NewRadioBox(mf, wx.ID_ANY, "Peer OS", wx.DefaultPosition, wx.DefaultSize, []string{"macOS", "Windows"}, 1, wx.HORIZONTAL)
	peerSizer.Add(radiobox1, 1, wx.ALL|wx.EXPAND, 5)

	// bottom half and big container
	bSizerBottom := wx.NewBoxSizer(wx.VERTICAL)
	bSizerTotal := wx.NewBoxSizer(wx.VERTICAL)

	// file selection box
	fileSizer := wx.NewBoxSizer(wx.HORIZONTAL)
	sendButton := wx.NewButton(mf, wx.ID_ANY, "Select File", wx.DefaultPosition, wx.DefaultSize, 0)
	receiveButton := wx.NewButton(mf, wx.ID_ANY, "Select Folder", wx.DefaultPosition, wx.DefaultSize, 0)
	receiveButton.Hide()
	fileBox := wx.NewTextCtrl(mf, wx.ID_ANY, "", wx.DefaultPosition, wx.DefaultSize, 0)
	fileSizer.Add(sendButton, 0, wx.ALL|wx.EXPAND, 5)
	fileSizer.Add(receiveButton, 0, wx.ALL|wx.EXPAND, 5)
	fileSizer.Add(fileBox, 1, wx.ALL|wx.EXPAND, 5)

	bSizerBottom.Add(fileSizer, 0, wx.ALL|wx.EXPAND, 5)

	radioSizer.Add(peerSizer, 1, wx.EXPAND, 5)
	modeSizer := wx.NewBoxSizer(wx.VERTICAL)

	radiobox2 := wx.NewRadioBox(mf, wx.ID_ANY, "Mode", wx.DefaultPosition, wx.DefaultSize, []string{"Send", "Receive"}, 1, wx.HORIZONTAL)
	modeSizer.Add(radiobox2, 1, wx.ALL|wx.EXPAND, 5)

	startButton := wx.NewButton(mf, wx.ID_ANY, "Start", wx.DefaultPosition, wx.DefaultSize, 0)
	bSizerBottom.Add(startButton, 0, wx.ALL|wx.EXPAND, 5)
	outputBox := wx.NewTextCtrl(mf, wx.ID_ANY, "", wx.DefaultPosition, wx.DefaultSize, wx.TE_MULTILINE|wx.TE_READONLY)
	outputBox.AppendText("Welcome to Flying Carpet!")

	progressBar := wx.NewGauge(mf, wx.ID_ANY, 100, wx.DefaultPosition, wx.DefaultSize, wx.GA_HORIZONTAL)
	progressBar.Hide()

	bSizerBottom.Add(outputBox, 1, wx.ALL|wx.EXPAND, 0)
	outputBox.SetSize(200, 200)
	bSizerBottom.Add(progressBar, 0, wx.ALL|wx.EXPAND, 5)

	radioSizer.Add(modeSizer, 1, wx.EXPAND, 5)

	bSizerTotal.Add(radioSizer, 0, wx.EXPAND, 5)
	bSizerTotal.Add(bSizerBottom, 1, wx.EXPAND, 5)

	// mode button action
	wx.Bind(mf, wx.EVT_RADIOBOX, func(e wx.Event) {
		if radiobox2.GetSelection() == 0 {
			receiveButton.Hide()
			sendButton.Show()
		} else if radiobox2.GetSelection() == 1 {
			sendButton.Hide()
			receiveButton.Show()
		}
		mf.Layout()
	}, radiobox2.GetId())

	// send button action
	wx.Bind(mf, wx.EVT_BUTTON, func(e wx.Event) {
		fd := wx.NewFileDialogT(wx.NullWindow, "Select Files", "", "", "*", wx.FD_OPEN, wx.DefaultPosition, wx.DefaultSize, "Open")
		if fd.ShowModal() != wx.ID_CANCEL {
			filename := fd.GetPath()
			fileBox.SetValue(filename)
		}
	}, sendButton.GetId())

	// receive button action
	wx.Bind(mf, wx.EVT_BUTTON, func(e wx.Event) {
		fd := wx.NewDirDialogT(wx.NullWindow, "Select Folder", "Open", wx.DD_DEFAULT_STYLE, wx.DefaultPosition, wx.DefaultSize)

		usr, err := user.Current()
		if err != nil {
			panic("Could not get user")
		}
		fd.SetPath(usr.HomeDir)

		if fd.ShowModal() != wx.ID_CANCEL {
			folder := fd.GetPath()
			fileBox.SetValue(folder + string(os.PathSeparator) + "file.out")
		}
	}, receiveButton.GetId())

	// start button action
	wx.Bind(mf, wx.EVT_BUTTON, func(e wx.Event) {
		mode, peer := "", ""
		if radiobox2.GetSelection() == 0 {
			mode = "send"
		} else if radiobox2.GetSelection() == 1 {
			mode = "receive"
		}
		if radiobox1.GetSelection() == 0 {
			peer = "mac"
		} else if radiobox1.GetSelection() == 1 {
			peer = "windows"
		}

		t := Transfer{
			Filepath:  fileBox.GetValue(),
			Port:      3290,
			Peer:      peer,
			AdHocChan: make(chan bool),
			Frame:     mf,
		}

		if mode == "send" {
			pd := wx.NewPasswordEntryDialog(mf, "Enter password from receiving end:", "", "", wx.OK|wx.CANCEL, wx.DefaultPosition)
			ret := pd.ShowModal()
			if ret == wx.ID_OK {
				_, err := os.Stat(t.Filepath)
				if err == nil {
					startButton.Enable(false)
					outputBox.AppendText("Entered password: " + pd.GetValue())
					t.Passphrase = pd.GetValue()
					// pd.Destroy()
					go t.mainRoutine(mode)
				} else {
					outputBox.AppendText("Could not find output file.")
				}
			} else {
				outputBox.AppendText("Password entry was cancelled.")
			}
		} else if mode == "receive" {
			_, err := os.Stat(t.Filepath)
			if err != nil {
				startButton.Enable(false)
				go t.mainRoutine(mode)
			} else {
				outputBox.AppendText("Error: destination file already exists.")
			}
		}
	}, startButton.GetId())

	// output box update event
	wx.Bind(mf, wx.EVT_THREAD, func(e wx.Event) {
		threadEvent := wx.ToThreadEvent(e)
		outputBox.AppendText("\n" + threadEvent.GetString())
	}, OUTPUT_BOX_UPDATE)

	// progress bar update event
	wx.Bind(mf, wx.EVT_THREAD, func(e wx.Event) {
		threadEvent := wx.ToThreadEvent(e)
		progressBar.SetValue(threadEvent.GetInt())
	}, PROGRESS_BAR_UPDATE)

	// progress bar display event
	wx.Bind(mf, wx.EVT_THREAD, func(e wx.Event) {
		progressBar.Show()
		mf.Layout()
	}, PROGRESS_BAR_SHOW)

	// start button enable event
	wx.Bind(mf, wx.EVT_THREAD, func(e wx.Event) {
		startButton.Enable(true)
		mf.Layout()
	}, START_BUTTON_ENABLE)

	mf.SetSizer(bSizerTotal)
	mf.Layout()
	mf.Centre(wx.BOTH)

	return mf
}

func (t *Transfer) output(msg string) {
	threadEvt := wx.NewThreadEvent(wx.EVT_THREAD, OUTPUT_BOX_UPDATE)
	threadEvt.SetString(msg)
	t.Frame.QueueEvent(threadEvt)
}

func (t *Transfer) enableStartButton() {
	startButtonEvt := wx.NewThreadEvent(wx.EVT_THREAD, START_BUTTON_ENABLE)
	t.Frame.QueueEvent(startButtonEvt)
}

type Transfer struct {
	Filepath    string
	Passphrase  string
	SSID        string
	Conn        net.Conn
	Port        int
	RecipientIP string
	Peer        string
	AdHocChan   chan bool
	Frame       *MainFrame
}

type Network interface {
	connectToPeer(*Transfer) bool
	getCurrentWifi() string
	resetWifi(*Transfer)
	teardown(*Transfer)
}

type WindowsNetwork struct {
	Mode         string // sending or receiving
	PreviousSSID string
}

type MacNetwork struct {
	Mode string // sending or receiving
}

type MainFrame struct {
	wx.Frame
	menuBar wx.MenuBar
}
