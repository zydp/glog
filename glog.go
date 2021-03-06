package glog

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

/*
Author: Yax.DAIPING
GitUrl: https://github.com/zydp
*/

/*
These flags define which text to prefix to each log entry generated by the Logger.
Bits are or'ed together to control what's printed.
There is no control over the order they appear (the order listed
here) or the format they present (as described in the comments).
The prefix is followed by a colon only when Llongfile or Lshortfile
is specified.
For example, flags Ldate | Ltime (or LstdFlags) produce,
	2009/01/23 01:23:23 message
while flags Ldate | Ltime | Lmicroseconds | Llongfile produce,
	2009/01/23 01:23:23.123123 /a/b/c/d.go:23: message
*/
const (
	Ldate         = 1 << iota     // the date in the local time zone: 2009/01/23
	Ltime                         // the time in the local time zone: 01:23:23
	Lmicroseconds                 // microsecond resolution: 01:23:23.123123.  assumes Ltime.
	Llongfile                     // full file name and line number: /a/b/c/d.go:23
	Lshortfile                    // final file name element and line number: d.go:23. overrides Llongfile
	LUTC                          // if Ldate or Ltime is set, use UTC rather than the local time zone
	LstdFlags     = Ldate | Ltime // initial values for the standard logger
)

const (
	SPLIT_FILE_SIZE    = 100 //the default file split size is 100MB
	TOTAL_ROTATE_SPLIT = 10  //the default total split count is 10

)

const (
	DEBUG = iota
	INFO
	WARNING
	ERROR
	FATAL
)

var (
	gStd     = newEx(os.Stderr, "", LstdFlags)                     //global handle
	levelStr = []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"} //Log level str
)

/*
A Logger represents an active logging object that generates lines of
output to an io.Writer. Each logging operation makes a single call to
the Writer's Write method. A Logger can be used simultaneously from
multiple goroutines; it guarantees to serialize access to the Writer.
*/
type Logger struct {
	mu               sync.Mutex // ensures atomic writes; protects the following fields
	prefix           string     // prefix to write at beginning of each line
	flag             int        // properties
	out              io.Writer  // destination for output
	buf              []byte     // for accumulating text to write
	filename         string     // log file name
	fileHandle       *os.File   // file handle
	writtenSize      uint64     // already written the size
	splitFileSize    uint64     // the logfile limit size
	splitRotateIndex int        // current rotate index
	totalRotateSplit int        // total rotate writes
}

/*
New creates a new Logger. The out variable sets the
destination to which log data will be written.
The prefix appears at the beginning of each generated log line.
The flag argument defines the logging properties.
The splitSize argument defines the logfile size, the unit is MB	  (1*1024*1024)Byte
The splitCount argument defines the total rotate split counts
*/

func New(filename string, prefix string, flag int) *Logger {
	return NewEx(filename, prefix, flag, SPLIT_FILE_SIZE, TOTAL_ROTATE_SPLIT)
}

func NewEx(filename string, prefix string, flag int, splitSize int, splitCount int) *Logger {
	openLogFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil
	}
	return &Logger{filename: filename, prefix: prefix, flag: flag, splitFileSize: uint64(splitSize * 1024 * 1024), totalRotateSplit: splitCount, fileHandle: openLogFile, out: openLogFile, writtenSize: 0}
}

func newEx(out io.Writer, prefix string, flag int) *Logger {
	return &Logger{filename: "", prefix: prefix, flag: flag, splitFileSize: uint64(SPLIT_FILE_SIZE * 1024 * 1024), totalRotateSplit: TOTAL_ROTATE_SPLIT, fileHandle: nil, out: out, writtenSize: 0}
}

/*rotate the log file*/
func (l *Logger) rotate() (err error) {
	_ = l.fileHandle.Close()
	_ = os.Rename(l.filename, fmt.Sprintf("%s.%d", l.filename, l.splitRotateIndex))
	l.fileHandle, err = os.OpenFile(l.filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	l.out = l.fileHandle
	l.splitRotateIndex++
	if l.splitRotateIndex > l.totalRotateSplit {
		l.splitRotateIndex = 0
	}
	return err
}

/*Set the file handle*/
func (l *Logger) setFileHandle(handle *os.File) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fileHandle = handle
}

/*SetOutput sets the output destination for the logger.*/
func (l *Logger) setOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = w
}

func SetOutput(w io.Writer) {
	gStd.mu.Lock()
	defer gStd.mu.Unlock()
	gStd.out = w
}

/*Cheap integer to fixed-width decimal ASCII. Give a negative width to avoid zero-padding.*/
func itoa(buf *[]byte, i int, wid int) {
	/*Assemble decimal in reverse order.*/
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

/*
formatHeader writes log header to buf in following order:
  * l.prefix (if it's not blank),
  * date and/or time (if corresponding flags are provided),
  * file and line number (if corresponding flags are provided).
*/
func (l *Logger) formatHeader(buf *[]byte, t time.Time, file string, line int) {
	*buf = append(*buf, l.prefix...)
	if l.flag&(Ldate|Ltime|Lmicroseconds) != 0 {
		if l.flag&LUTC != 0 {
			t = t.UTC()
		}
		if l.flag&Ldate != 0 {
			year, month, day := t.Date()
			itoa(buf, year, 4)
			*buf = append(*buf, '/')
			itoa(buf, int(month), 2)
			*buf = append(*buf, '/')
			itoa(buf, day, 2)
			*buf = append(*buf, ' ')
		}
		if l.flag&(Ltime|Lmicroseconds) != 0 {
			hour, min, sec := t.Clock()
			itoa(buf, hour, 2)
			*buf = append(*buf, ':')
			itoa(buf, min, 2)
			*buf = append(*buf, ':')
			itoa(buf, sec, 2)
			if l.flag&Lmicroseconds != 0 {
				*buf = append(*buf, '.')
				itoa(buf, t.Nanosecond()/1e3, 6)
			}
			*buf = append(*buf, ' ')
		}
	}
	if l.flag&(Lshortfile|Llongfile) != 0 {
		if l.flag&Lshortfile != 0 {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		*buf = append(*buf, file...)
		*buf = append(*buf, ':')
		itoa(buf, line, -1)
		*buf = append(*buf, ": "...)
	}
}

/*
Output writes the output for a logging event. The string s contains
the text to print after the prefix specified by the flags of the
Logger. A newline is appended if the last character of s is not
already a newline. Calldepth is used to recover the PC and is
provided for generality, although at the moment on all pre-defined
paths it will be 2.
*/
func (l *Logger) Output(calldepth int, s string) error {
	now := time.Now() // get this early.
	var file string
	var line int
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.flag&(Lshortfile|Llongfile) != 0 {
		/*Release lock while getting caller info - it's expensive.*/
		l.mu.Unlock()
		var ok bool
		_, file, line, ok = runtime.Caller(calldepth)
		if !ok {
			file = "???"
			line = 0
		}
		l.mu.Lock()
	}
	l.buf = l.buf[:0]
	l.formatHeader(&l.buf, now, file, line)
	l.buf = append(l.buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}
	n, err := l.out.Write(l.buf)
	l.writtenSize += uint64(n)
	if l.writtenSize >= l.splitFileSize {
		if l.filename != "" {
			l.rotate()
		}
		l.writtenSize = 0
	}
	return err
}
func Output(calldepth int, s string) error {
	return gStd.Output(calldepth+1, s) // +1 for this frame.
}

/*#################### S u g a r #####################*/
func (l *Logger) Debug(format string, v ...interface{}) {
	l.Output(2, fmt.Sprintf(fmt.Sprintf("[%s]:%s", levelStr[DEBUG], format), v...))
}
func Debug(format string, v ...interface{}) {
	gStd.Output(2, fmt.Sprintf(fmt.Sprintf("[%s]:%s", levelStr[DEBUG], format), v...))
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.Output(2, fmt.Sprintf(fmt.Sprintf("[%s]:%s", levelStr[INFO], format), v...))
}
func Info(format string, v ...interface{}) {
	gStd.Output(2, fmt.Sprintf(fmt.Sprintf("[%s]:%s", levelStr[INFO], format), v...))
}

func (l *Logger) Warn(format string, v ...interface{}) {
	l.Output(2, fmt.Sprintf(fmt.Sprintf("[%s]:%s", levelStr[WARNING], format), v...))
}
func Warn(format string, v ...interface{}) {
	gStd.Output(2, fmt.Sprintf(fmt.Sprintf("[%s]:%s", levelStr[WARNING], format), v...))
}

func (l *Logger) Err(format string, v ...interface{}) {
	l.Output(2, fmt.Sprintf(fmt.Sprintf("[%s]:%s", levelStr[ERROR], format), v...))
}
func Err(format string, v ...interface{}) {
	gStd.Output(2, fmt.Sprintf(fmt.Sprintf("[%s]:%s", levelStr[ERROR], format), v...))
}

/*
Printf calls l.Output to print to the logger.
Arguments are handled in the manner of fmt.Printf.
*/
func (l *Logger) Printf(format string, v ...interface{}) {
	l.Output(2, fmt.Sprintf(format, v...))
}
func Printf(format string, v ...interface{}) {
	gStd.Output(2, fmt.Sprintf(format, v...))
}

/*
Print calls l.Output to print to the logger.
Arguments are handled in the manner of fmt.Print.
*/
func (l *Logger) Print(v ...interface{}) { l.Output(2, fmt.Sprint(v...)) }
func Print(v ...interface{}) {
	gStd.Output(2, fmt.Sprint(v...))
}

/*
Println calls l.Output to print to the logger.
Arguments are handled in the manner of fmt.Println.
*/
func (l *Logger) Println(v ...interface{}) { l.Output(2, fmt.Sprintln(v...)) }
func Println(v ...interface{}) {
	gStd.Output(2, fmt.Sprintln(v...))
}

/*Fatal is equivalent to l.Print() followed by a call to os.Exit(1).*/
func (l *Logger) Fatal(v ...interface{}) {
	l.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}
func Fatal(v ...interface{}) {
	gStd.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

/*Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).*/
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}
func Fatalf(format string, v ...interface{}) {
	gStd.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}

/*Fatalln is equivalent to l.Println() followed by a call to os.Exit(1).*/
func (l *Logger) Fatalln(v ...interface{}) {
	l.Output(2, fmt.Sprintln(v...))
	os.Exit(1)
}
func Fatalln(v ...interface{}) {
	gStd.Output(2, fmt.Sprintln(v...))
	os.Exit(1)
}

/*Panic is equivalent to l.Print() followed by a call to panic().*/
func (l *Logger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.Output(2, s)
	panic(s)
}
func Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	gStd.Output(2, s)
	panic(s)
}

/*Panicf is equivalent to l.Printf() followed by a call to panic().*/
func (l *Logger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.Output(2, s)
	panic(s)
}
func Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	gStd.Output(2, s)
	panic(s)
}

/*Panicln is equivalent to l.Println() followed by a call to panic().*/
func (l *Logger) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	l.Output(2, s)
	panic(s)
}
func Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	gStd.Output(2, s)
	panic(s)
}

/*Flags returns the output flags for the logger.*/
func (l *Logger) Flags() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.flag
}
func Flags() int {
	return gStd.Flags()
}

/*SetFlags sets the output flags for the logger.*/
func (l *Logger) SetFlags(flag int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flag = flag
}

func SetFlags(flag int) {
	gStd.SetFlags(flag)
}

/*Prefix returns the output prefix for the logger.*/
func (l *Logger) Prefix() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.prefix
}
func Prefix() string {
	return gStd.Prefix()
}

/*SetPrefix sets the output prefix for the logger.*/
func (l *Logger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

func SetPrefix(prefix string) {
	gStd.SetPrefix(prefix)
}

// Writer returns the output destination for the logger.
func (l *Logger) Writer() io.Writer {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.out
}
func Writer() io.Writer {
	return gStd.Writer()
}
