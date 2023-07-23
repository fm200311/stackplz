package event

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "stackplz/pkg/util"
    "stackplz/user/config"
    "strings"
    "time"
)

type ContextEvent struct {
    CommonEvent
    ts            uint64
    eventid       uint32
    host_tid      uint32
    host_pid      uint32
    tid           uint32
    pid           uint32
    uid           uint32
    comm          [16]byte
    argnum        uint8
    padding       [7]byte
    part_raw_size uint32
}

func (this *ContextEvent) NewSyscallEvent() IEventStruct {
    event := &SyscallEvent{ContextEvent: *this}
    err := event.ParseContext()
    if err != nil {
        panic(fmt.Sprintf("NewMmap2Event.ParseContext() err:%v", err))
    }
    return event
}

func (this *ContextEvent) NewUprobeEvent() IEventStruct {
    event := &UprobeEvent{ContextEvent: *this}
    err := event.ParseContext()
    if err != nil {
        panic(fmt.Sprintf("NewUprobeEvent.ParseContext() err:%v", err))
    }
    return event
}

func (this *ContextEvent) Decode() (err error) {
    return nil
}

func (this *ContextEvent) String() string {
    var s string
    s = fmt.Sprintf("[ContextEvent] %s eventid:%d argnum:%d time:%d", this.GetUUID(), this.eventid, this.argnum, this.ts)
    return s
}

func (this *ContextEvent) GetUUID() string {
    return fmt.Sprintf("%d_%d", this.pid, this.tid)
}

func (this *ContextEvent) GetEventId() uint32 {
    return this.eventid
}

type Arg_raw_size struct {
    Index       uint8
    PartRawSize uint32
}

func (this *ContextEvent) ParsePadding() (err error) {
    // PERF_SAMPLE_RAW 末尾可能包含 padding 这里先把
    // ... nr/probe_index|lr|pc|sp|args...|size_before|padding
    var arg Arg_raw_size
    if err = binary.Read(this.buf, binary.LittleEndian, &arg); err != nil {
        panic(fmt.Sprintf("binary.Read err:%v", err))
    }
    // 处理掉 padding
    this.part_raw_size = arg.PartRawSize
    padding_size := 4 - (arg.PartRawSize+uint32(binary.Size(arg)))%4
    if padding_size != 4 {
        payload := make([]byte, padding_size)
        if err = binary.Read(this.buf, binary.LittleEndian, &payload); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
    }
    // if this.mconf.Debug {
    //     this.logger.Printf("PartRawSize:%d padding_size:%d", arg.PartRawSize, padding_size)
    // }
    return nil
}

func (this *ContextEvent) ParseContext() (err error) {
    this.buf = bytes.NewBuffer(this.rec.RawSample)
    if err = binary.Read(this.buf, binary.LittleEndian, &this.ts); err != nil {
        panic(fmt.Sprintf("binary.Read err:%v", err))
    }
    if err = binary.Read(this.buf, binary.LittleEndian, &this.eventid); err != nil {
        panic(fmt.Sprintf("binary.Read err:%v", err))
    }
    if err = binary.Read(this.buf, binary.LittleEndian, &this.host_tid); err != nil {
        panic(fmt.Sprintf("binary.Read err:%v", err))
    }
    if err = binary.Read(this.buf, binary.LittleEndian, &this.host_pid); err != nil {
        panic(fmt.Sprintf("binary.Read err:%v", err))
    }
    if err = binary.Read(this.buf, binary.LittleEndian, &this.tid); err != nil {
        panic(fmt.Sprintf("binary.Read err:%v", err))
    }
    if err = binary.Read(this.buf, binary.LittleEndian, &this.pid); err != nil {
        panic(fmt.Sprintf("binary.Read err:%v", err))
    }
    if err = binary.Read(this.buf, binary.LittleEndian, &this.uid); err != nil {
        panic(fmt.Sprintf("binary.Read err:%v", err))
    }
    if err = binary.Read(this.buf, binary.LittleEndian, &this.comm); err != nil {
        panic(fmt.Sprintf("binary.Read err:%v", err))
    }
    if err = binary.Read(this.buf, binary.LittleEndian, &this.argnum); err != nil {
        panic(fmt.Sprintf("binary.Read err:%v", err))
    }
    if err = binary.Read(this.buf, binary.LittleEndian, &this.padding); err != nil {
        panic(fmt.Sprintf("binary.Read err:%v", err))
    }
    return nil
}

func (this *ContextEvent) Clone() IEventStruct {
    event := new(ContextEvent)
    // event.event_type = EventTypeSysCallData
    return event
}

func (this *ContextEvent) ParseArgByType(point_arg *config.PointArg, ptr Arg_reg) {
    var err error
    if ptr.Address == 0 {
        point_arg.AppendValue("(NULL)")
        return
    }
    // 这个函数先处理基础类型

    if point_arg.Type == config.TYPE_POINTER {
        // 对于指针类型 需要先处理
        var next_ptr Arg_reg
        if err = binary.Read(this.buf, binary.LittleEndian, &next_ptr); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        // if this.mconf.Debug {
        //     this.logger.Printf("[buf] len:%d cap:%d off:%d", this.buf.Len(), this.buf.Cap(), this.buf.Cap()-this.buf.Len())
        // }
        if next_ptr.Address == 0 {
            point_arg.AppendValue("(0x0)")
            return
        }
        point_arg.AppendValue(fmt.Sprintf("(*0x%x)%s", next_ptr.Address, this.ParseArg(point_arg, next_ptr)))
        return
    } else {
        // 这种一般就是特殊类型 获取结构体了
        point_arg.AppendValue(this.ParseArg(point_arg, ptr))
    }
}
func (this *ContextEvent) ParseArg(point_arg *config.PointArg, ptr Arg_reg) string {
    var err error
    switch point_arg.AliasType {
    case config.TYPE_NONE:
        panic("AliasType TYPE_NONE can not be here")
    case config.TYPE_NUM:
        panic("AliasType TYPE_NUM can not be here")
    case config.TYPE_BUFFER_T:
        var arg Arg_Buffer_t
        if err = binary.Read(this.buf, binary.LittleEndian, &arg.Arg_str); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        payload := make([]byte, arg.Len)
        if err = binary.Read(this.buf, binary.LittleEndian, &payload); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        arg.Payload = payload
        if this.mconf.DumpHex {
            return arg.HexFormat(this.mconf.Color)
        } else {
            return arg.Format()
        }
    case config.TYPE_STRING:
        var arg Arg_str
        if err = binary.Read(this.buf, binary.LittleEndian, &arg); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        payload := make([]byte, arg.Len)
        if err = binary.Read(this.buf, binary.LittleEndian, &payload); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return fmt.Sprintf("(%s)", util.B2STrim(payload))
    case config.TYPE_STRING_ARR:
        var arg_str_arr Arg_str_arr
        if err = binary.Read(this.buf, binary.LittleEndian, &arg_str_arr); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        var str_arr []string
        for i := 0; i < int(arg_str_arr.Count); i++ {
            var len uint32
            if err = binary.Read(this.buf, binary.LittleEndian, &len); err != nil {
                panic(fmt.Sprintf("binary.Read err:%v", err))
            }
            payload := make([]byte, len)
            if err = binary.Read(this.buf, binary.LittleEndian, &payload); err != nil {
                panic(fmt.Sprintf("binary.Read err:%v", err))
            }
            str_arr = append(str_arr, util.B2STrim(payload))
        }
        return fmt.Sprintf("[%s]", strings.Join(str_arr, ", "))
    case config.TYPE_POINTER:
        // 先解析参数寄存器本身的值
        var ptr_value Arg_reg
        // 再解析参数寄存器指向地址的值
        if err = binary.Read(this.buf, binary.LittleEndian, &ptr_value); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return fmt.Sprintf("(0x%x)", ptr_value.Address)
    case config.TYPE_SIGSET:
        var sigs [8]uint32
        if err = binary.Read(this.buf, binary.LittleEndian, &sigs); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        var fmt_sigs []string
        for i := 0; i < len(sigs); i++ {
            fmt_sigs = append(fmt_sigs, fmt.Sprintf("0x%x", sigs[i]))
        }
        return fmt.Sprintf("(sigs=[%s])", strings.Join(fmt_sigs, ","))
    case config.TYPE_POLLFD:
        var pollfd Arg_Pollfd
        if err = binary.Read(this.buf, binary.LittleEndian, &pollfd); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return fmt.Sprintf("(fd=%d, events=%d, revents=%d)", pollfd.Fd, pollfd.Events, pollfd.Revents)
    case config.TYPE_STRUCT:
        payload := make([]byte, point_arg.Size)
        if err = binary.Read(this.buf, binary.LittleEndian, &payload); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return fmt.Sprintf("([hex]%x)", payload)
    case config.TYPE_TIMEZONE:
        var arg Arg_TimeZone_t
        if err = binary.Read(this.buf, binary.LittleEndian, &arg); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg.Format()
    case config.TYPE_PTHREAD_ATTR:
        var arg Arg_Pthread_attr_t
        if err = binary.Read(this.buf, binary.LittleEndian, &arg); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg.Format()
    case config.TYPE_TIMEVAL:
        var arg Arg_Timeval
        if err = binary.Read(this.buf, binary.LittleEndian, &arg); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg.Format()
    case config.TYPE_TIMESPEC:
        var arg Arg_Timespec
        if err = binary.Read(this.buf, binary.LittleEndian, &arg); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg.Format()
    case config.TYPE_STAT:
        var arg_stat_t Arg_Stat_t
        if err = binary.Read(this.buf, binary.LittleEndian, &arg_stat_t); err != nil {
            this.logger.Printf("ContextEvent eventid:%d RawSample:\n%s", this.eventid, util.HexDump(this.rec.RawSample, util.COLORRED))
            time.Sleep(3 * 1000 * time.Millisecond)
            panic(fmt.Sprintf("binary.Read %s err:%v", util.B2STrim(this.comm[:]), err))
        }
        return arg_stat_t.Format()
    case config.TYPE_STATFS:
        var arg_statfs_t Arg_Statfs_t
        if err = binary.Read(this.buf, binary.LittleEndian, &arg_statfs_t); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg_statfs_t.Format()
    case config.TYPE_SIGACTION:
        var arg_sigaction Arg_Sigaction
        if err = binary.Read(this.buf, binary.LittleEndian, &arg_sigaction); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg_sigaction.Format()
    case config.TYPE_UTSNAME:
        var arg_name Arg_Utsname
        if err = binary.Read(this.buf, binary.LittleEndian, &arg_name); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        sysname := B2S(arg_name.Sysname[:])
        nodename := B2S(arg_name.Nodename[:])
        release := B2S(arg_name.Release[:])
        version := B2S(arg_name.Version[:])
        machine := B2S(arg_name.Machine[:])
        domainname := B2S(arg_name.Domainname[:])
        return fmt.Sprintf("{sysname=%s, nodename=%s, release=%s, version=%s, machine=%s, domainname=%s}", sysname, nodename, release, version, machine, domainname)
    case config.TYPE_SOCKADDR:
        var arg Arg_RawSockaddrUnix
        if err = binary.Read(this.buf, binary.LittleEndian, &arg); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg.Format()
    case config.TYPE_RUSAGE:
        var arg_rusage Arg_Rusage
        if err = binary.Read(this.buf, binary.LittleEndian, &arg_rusage); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg_rusage.Format()
    case config.TYPE_IOVEC:
        var arg Arg_Iovec_t
        if err = binary.Read(this.buf, binary.LittleEndian, &arg.Arg_Iovec); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        payload := make([]byte, arg.BufLen)
        if err = binary.Read(this.buf, binary.LittleEndian, &payload); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        arg.Payload = payload
        return arg.Format()
    case config.TYPE_EPOLLEVENT:
        var arg_epollevent Arg_EpollEvent
        if err = binary.Read(this.buf, binary.LittleEndian, &arg_epollevent); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg_epollevent.Format()
    case config.TYPE_SYSINFO:
        var arg Arg_Sysinfo_t
        if err = binary.Read(this.buf, binary.LittleEndian, &arg); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg.Format()
    case config.TYPE_SIGINFO:
        // 这个读取出来有问题
        var arg Arg_SigInfo
        if err = binary.Read(this.buf, binary.LittleEndian, &arg); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg.Format()
    case config.TYPE_MSGHDR:
        var arg Arg_Msghdr
        if err = binary.Read(this.buf, binary.LittleEndian, &arg); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg.Format()
    case config.TYPE_ITIMERSPEC:
        var arg Arg_ItTmerspec
        if err = binary.Read(this.buf, binary.LittleEndian, &arg); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg.Format()
    case config.TYPE_STACK_T:
        var arg Arg_Stack_t
        if err = binary.Read(this.buf, binary.LittleEndian, &arg); err != nil {
            panic(fmt.Sprintf("binary.Read err:%v", err))
        }
        return arg.Format()
    default:
        panic(fmt.Sprintf("unknown point_arg.AliasType %d", point_arg.AliasType))
    }
}
