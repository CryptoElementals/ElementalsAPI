package log

import (
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalLogger *Logger
)

// init a development logger if no config have
func init() {
	l, err := NewLogger(&Config{
		Development: true,
	})
	if err != nil {
		panic(err)
	}
	globalLogger = l
}

type Config struct {
	Dir          string `mapstructure:"dir"`
	Prefix       string `mapstructure:"prefix"`
	Suffix       string `mapstructure:"suffix"`
	Level        string `mapstructure:"level"` // log level
	MaxAge       int    `mapstructure:"maxAge"`
	RotationTime int    `mapstructure:"rotationTime"`
	Development  bool   `mapstructure:"development"`
}

func InitGlobalLogger(cfg *Config, opts ...zap.Option) error {
	log, err := NewLogger(cfg, opts...)
	globalLogger = log
	return err
}

func GlobalLogger() *Logger {
	return globalLogger
}

func FromZap(logger *zap.Logger) *Logger {
	return &Logger{
		SugaredLogger: logger.Sugar(),
	}
}

func DPanic(args ...interface{}) {
	globalLogger.DPanic(args...)
}

func DPanicf(template string, args ...interface{}) {
	globalLogger.DPanicf(template, args...)
}

func DPanicw(msg string, keysAndValues ...interface{}) {
	globalLogger.DPanicw(msg, keysAndValues...)
}

func Debug(args ...interface{}) {
	globalLogger.Debug(args...)
}

func Debugf(template string, args ...interface{}) {
	globalLogger.Debugf(template, args...)
}

func Debugw(msg string, keysAndValues ...interface{}) {
	globalLogger.Debugw(msg, keysAndValues...)
}

func Error(args ...interface{}) {
	globalLogger.Error(args...)
}

func Errorf(template string, args ...interface{}) {
	globalLogger.Errorf(template, args...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	globalLogger.Errorw(msg, keysAndValues...)
}

func Fatal(args ...interface{}) {
	globalLogger.Fatal(args...)
}

func Fatalf(template string, args ...interface{}) {
	globalLogger.Fatalf(template, args...)
}

func Fatalw(msg string, keysAndValues ...interface{}) {
	globalLogger.Fatalw(msg, keysAndValues...)
}

func Info(args ...interface{}) {
	globalLogger.Info(args...)
}

func Infof(template string, args ...interface{}) {
	globalLogger.Infof(template, args...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	globalLogger.Infow(msg, keysAndValues...)
}

func Panic(args ...interface{}) {
	globalLogger.Panic(args...)
}

func Panicf(template string, args ...interface{}) {
	globalLogger.Panicf(template, args...)
}

func Panicw(msg string, keysAndValues ...interface{}) {
	globalLogger.Panicw(msg, keysAndValues...)
}

func Warn(args ...interface{}) {
	globalLogger.Warn(args...)
}

func Warnf(template string, args ...interface{}) {
	globalLogger.Warnf(template, args...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	globalLogger.Warnw(msg, keysAndValues...)
}

const (
	defaultLogDir         = "/var/log/"
	defaultLogPrefix      = "ufile-api"
	defaultLogSuffix      = ".log"
	defaultMaxAge         = 7 // day
	defaultRotationTime   = 2 // hour
	defaultLogLevelString = "DEBUG"
)

type Logger struct {
	*zap.SugaredLogger
	w io.Writer
}

func NewLogger(cfg *Config, opts ...zap.Option) (*Logger, error) {
	logger := &Logger{}
	dir := cfg.Dir
	prefix := cfg.Prefix
	suffix := cfg.Suffix
	maxAge := cfg.MaxAge
	rotationTime := cfg.RotationTime
	level := cfg.Level
	development := cfg.Development

	if dir == "" {
		dir = defaultLogDir
	}
	if prefix == "" {
		prefix = defaultLogPrefix
	}
	if suffix == "" {
		suffix = defaultLogSuffix
	}
	if maxAge <= 0 {
		maxAge = defaultMaxAge
	}
	if rotationTime <= 0 {
		rotationTime = defaultRotationTime
	}
	if level == "" {
		level = defaultLogLevelString
	}

	writer, err := NewRotateLogger(dir, prefix, suffix, maxAge, rotationTime, development)
	if err != nil {
		return nil, err
	}
	ws := zapcore.AddSync(writer)
	var zl zapcore.Level
	zl, err = zapcore.ParseLevel(level)
	if err != nil {
		zl = zap.ErrorLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		ws,
		zl,
	)

	allCore := zapcore.NewTee(core)
	l := zap.New(allCore, zap.AddCaller())

	if development {
		l = l.WithOptions(zap.Development())
	}

	if len(opts) != 0 {
		l = l.WithOptions(opts...)
	}

	l.WithOptions()

	logger.SugaredLogger = l.Sugar()
	return logger, nil
}

func (l *Logger) L() *zap.Logger {
	if nil == l.SugaredLogger {
		return zap.L()
	}
	return l.SugaredLogger.Desugar()
}

func (l *Logger) S() *zap.SugaredLogger {
	if nil == l.SugaredLogger {
		return zap.S()
	}
	return l.SugaredLogger
}

func (l *Logger) Named(label string) *Logger {
	return &Logger{
		SugaredLogger: l.SugaredLogger.Named(label),
	}
}

func (l *Logger) Writer() io.Writer {
	return l.w
}

func NewRotateLogger(dir, prefix, suffix string, maxAge, rotationTime int, development bool) (io.Writer, error) {
	if development {
		return zapcore.Lock(os.Stdout), nil
	}
	if dir[0:1] != "/" {
		dir = getCurrPath() + "/" + dir
	}
	if dir[len(dir)-1:] == "/" {
		dir = dir[:len(dir)-1]
	}
	// 校验路径是否存在。如果不存在，则创建之
	if ok, err := pathExist(dir); err != nil {
		return nil, err
	} else if !ok {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	baseLogName := prefix + "." + suffix
	baseLogPath := path.Join(dir, baseLogName)
	writer, err := rotatelogs.New(
		baseLogPath+".%Y%m%d%H%M",
		rotatelogs.WithLinkName(baseLogPath),                               // 生成软链，指向最新日志文件
		rotatelogs.WithMaxAge(time.Duration(maxAge)*24*time.Hour),          // 文件最大保存时间
		rotatelogs.WithRotationTime(time.Duration(rotationTime)*time.Hour), // 日志切割时间间隔
	)
	if err != nil {
		return nil, err
	}

	return writer, nil
}

func getCurrPath() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	index := strings.LastIndex(path, string(os.PathSeparator))
	ret := path[:index]
	return ret
}

func pathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func StringField(key, val string) zapcore.Field {
	return zapcore.Field{
		Key:    key,
		Type:   zapcore.StringType,
		String: val,
	}
}
