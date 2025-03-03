// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/nazaerrors"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
)

type PullSessionObserver interface {
	BaseInSessionObserver
}

type PullSessionOption struct {
	// 从调用Pull函数，到接收音视频数据的前一步，也即收到rtsp play response的超时时间
	// 如果为0，则没有超时时间
	PullTimeoutMs int

	OverTcp bool // 是否使用interleaved模式，也即是否通过rtsp command tcp连接传输rtp/rtcp数据
}

var defaultPullSessionOption = PullSessionOption{
	PullTimeoutMs: 10000,
	OverTcp:       false,
}

type PullSession struct {
	uniqueKey     string // const after ctor
	cmdSession    *ClientCommandSession
	baseInSession *BaseInSession
}

type ModPullSessionOption func(option *PullSessionOption)

func NewPullSession(observer PullSessionObserver, modOptions ...ModPullSessionOption) *PullSession {
	option := defaultPullSessionOption
	for _, fn := range modOptions {
		fn(&option)
	}

	uk := base.GenUkRtspPullSession()
	s := &PullSession{
		uniqueKey: uk,
	}
	cmdSession := NewClientCommandSession(CcstPullSession, uk, s, func(opt *ClientCommandSessionOption) {
		opt.DoTimeoutMs = option.PullTimeoutMs
		opt.OverTcp = option.OverTcp
	})
	baseInSession := NewBaseInSessionWithObserver(uk, s, observer)
	s.baseInSession = baseInSession
	s.cmdSession = cmdSession
	nazalog.Infof("[%s] lifecycle new rtsp PullSession. session=%p", uk, s)
	return s
}

// 阻塞直到和对端完成拉流前，握手部分的工作（也即收到RTSP Play response），或者发生错误
func (session *PullSession) Pull(rawUrl string) error {
	nazalog.Debugf("[%s] pull. url=%s", session.uniqueKey, rawUrl)
	return session.cmdSession.Do(rawUrl)
}

func (session *PullSession) GetSdp() ([]byte, sdp.LogicContext) {
	return session.baseInSession.GetSdp()
}

// 文档请参考： interface IClientSessionLifecycle
func (session *PullSession) Dispose() error {
	nazalog.Infof("[%s] lifecycle dispose rtsp PullSession. session=%p", session.uniqueKey, session)
	e1 := session.cmdSession.Dispose()
	e2 := session.baseInSession.Dispose()
	return nazaerrors.CombineErrors(e1, e2)
}

// 文档请参考： interface IClientSessionLifecycle
func (session *PullSession) WaitChan() <-chan error {
	return session.cmdSession.WaitChan()
}

// 文档请参考： interface ISessionUrlContext
func (session *PullSession) Url() string {
	return session.cmdSession.Url()
}

// 文档请参考： interface ISessionUrlContext
func (session *PullSession) AppName() string {
	return session.cmdSession.AppName()
}

// 文档请参考： interface ISessionUrlContext
func (session *PullSession) StreamName() string {
	return session.cmdSession.StreamName()
}

// 文档请参考： interface ISessionUrlContext
func (session *PullSession) RawQuery() string {
	return session.cmdSession.RawQuery()
}

// 文档请参考： interface IObject
func (session *PullSession) UniqueKey() string {
	return session.uniqueKey
}

// 文档请参考： interface ISessionStat
func (session *PullSession) GetStat() base.StatSession {
	stat := session.baseInSession.GetStat()
	stat.RemoteAddr = session.cmdSession.RemoteAddr()
	return stat
}

// 文档请参考： interface ISessionStat
func (session *PullSession) UpdateStat(intervalSec uint32) {
	session.baseInSession.UpdateStat(intervalSec)
}

// 文档请参考： interface ISessionStat
func (session *PullSession) IsAlive() (readAlive, writeAlive bool) {
	return session.baseInSession.IsAlive()
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PullSession) OnConnectResult() {
	// noop
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PullSession) OnDescribeResponse(rawSdp []byte, sdpLogicCtx sdp.LogicContext) {
	session.baseInSession.InitWithSdp(rawSdp, sdpLogicCtx)
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PullSession) OnSetupWithConn(uri string, rtpConn, rtcpConn *nazanet.UdpConnection) {
	_ = session.baseInSession.SetupWithConn(uri, rtpConn, rtcpConn)
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PullSession) OnSetupWithChannel(uri string, rtpChannel, rtcpChannel int) {
	_ = session.baseInSession.SetupWithChannel(uri, rtpChannel, rtcpChannel)
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PullSession) OnSetupResult() {
	session.baseInSession.WriteRtpRtcpDummy()
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PullSession) OnInterleavedPacket(packet []byte, channel int) {
	session.baseInSession.HandleInterleavedPacket(packet, channel)
}

// IInterleavedPacketWriter, callback by BaseInSession
func (session *PullSession) WriteInterleavedPacket(packet []byte, channel int) error {
	return session.cmdSession.WriteInterleavedPacket(packet, channel)
}
