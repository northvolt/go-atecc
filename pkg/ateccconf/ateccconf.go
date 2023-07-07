package ateccconf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
)

// TODO: add easier handling of bits

// Default608 is an example configuration for ATECC608A.
//
// First 16 bytes as expected from a normal configuration is not included.
// These are fixed by the factory.
var Default608 = []byte{
	0x6a, 0x00, 0x00, 0x01, 0x85, 0x00, 0x82, 0x00, 0x85, 0x20, 0x85, 0x20, 0x85, 0x20, 0xc6, 0x46,
	0x8f, 0x0f, 0x9f, 0x8f, 0x0f, 0x0f, 0x8f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f,
	0x0d, 0x1f, 0x0f, 0x0f, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xf7, 0x00, 0x69, 0x76, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0x55, 0xff, 0xff, 0x0e, 0x60, 0x00, 0x00, 0x00, 0x00,
	0x53, 0x00, 0x53, 0x00, 0x73, 0x00, 0x73, 0x00, 0x73, 0x00, 0x38, 0x00, 0x7c, 0x00, 0x1c, 0x00,
	0x3c, 0x00, 0x1a, 0x00, 0x3c, 0x00, 0x30, 0x00, 0x3c, 0x00, 0x30, 0x00, 0x12, 0x00, 0x30, 0x00,
}

func DefaultConfig608() *Config608 {
	var conf Config608
	err := UnmarshalPartial(Default608, PermanentOffset608, &conf)
	if err != nil {
		panic(err)
	}
	return &conf
}

type AESEnable struct {
	// Bits contains of
	// * enabled 1
	// * reserved 7
	Bits uint8
}

type aesEnabledBits struct {
	Enabled  bool `json:"enabled"`
	Reserved byte `json:"reserved"`
}

func (a AESEnable) Enabled() bool {
	return a.Bits&0x01 != 0
}

func (a AESEnable) Reserved() byte {
	return a.Bits >> 1
}

func (a AESEnable) MarshalJSON() ([]byte, error) {
	return json.Marshal(aesEnabledBits{
		Enabled:  a.Enabled(),
		Reserved: a.Reserved(),
	})
}

type I2CEnable struct {
	// Bits contains of
	// * Enabled 1
	// * Reserved 7
	Bits uint8
}

type i2cEnableBits struct {
	Enabled  bool `json:"enabled"`
	Reserved byte `json:"reserved"`
}

func (i I2CEnable) Enabled() bool {
	return i.Bits&0x01 != 0
}

func (i I2CEnable) Reserved() byte {
	return i.Bits >> 1
}

func (i I2CEnable) MarshalJSON() ([]byte, error) {
	return json.Marshal(i2cEnableBits{
		Enabled:  i.Enabled(),
		Reserved: i.Reserved(),
	})
}

type CountMatch struct {
	// Bits contains of:
	// * Enabled       1
	// * Reserved      3
	// * CountMatchKey 4
	Bits uint8
}

type countMatchBits struct {
	Enabled  bool `json:"enabled"`
	Reserved byte `json:"reserved"`
	Key      byte `json:"key"`
}

func (cm CountMatch) Enabled() bool {
	return cm.Bits&0x01 != 0
}

func (cm CountMatch) Reserved() byte {
	return (cm.Bits & 0x0e) >> 1
}

func (cm CountMatch) Key() byte {
	return (cm.Bits & 0xf0) >> 4
}

func (cm CountMatch) MarshalJSON() ([]byte, error) {
	return json.Marshal(countMatchBits{
		Enabled:  cm.Enabled(),
		Reserved: cm.Reserved(),
		Key:      cm.Key(),
	})
}

const (
	// ChipModeOffset is the byte offset within the configuration zone
	ChipModeOffset = 19

	// PermanentOffset608 is the device offset which cannot be written to.
	PermanentOffset608 = 16

	LockOffsetBlock = 2
	LockOffsetWord  = 5

	// LockOffset is the byte offset to the lock bytes.
	//
	// Note: this offset is bigger than one block size.
	LockOffset = LockOffsetBlock*32 + LockOffsetWord*4
)

type ClockDivider uint8

const (
	// ClockDividerM0 is high speed.
	ClockDividerM0 = ClockDivider(0x00 >> 3)
	ClockDividerM1 = ClockDivider(0x28 >> 3)
	ClockDividerM2 = ClockDivider(0x68 >> 3)
)

func (c ClockDivider) String() string {
	switch c {
	case ClockDividerM0:
		return "m0"
	case ClockDividerM1:
		return "m1"
	case ClockDividerM2:
		return "m2"
	default:
		return "unknown"
	}
}

func (c ClockDivider) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

type ChipMode608 struct {
	// Bits consists of:
	// * UserExtraAdd     1
	//   1 Alternate I2C address mode is enabled
	// * TTLenable        1
	//   0 I/O’s use Fixed Reference mode
	// * WatchdogDuration 1
	//   0 Watchdog Time is set to 1.3s
	// * ClockDivider     5
	Bits uint8
}

type chipMode608Bits struct {
	UserExtraAdd     bool         `json:"user_extra_add"`
	TTLEnabled       bool         `json:"ttl_enabled"`
	WatchdogDuration bool         `json:"watchdog_duration"`
	ClockDivider     ClockDivider `json:"clock_divider"`
}

func (cm ChipMode608) UserExtraAdd() bool {
	return cm.Bits&0x01 != 0
}

func (cm ChipMode608) TTLEnabled() bool {
	return (cm.Bits & 0x02) != 0
}

func (cm ChipMode608) WatchdogDuration() bool {
	return (cm.Bits & 0x04) != 0
}

func (cm ChipMode608) ClockDivider() ClockDivider {
	return ClockDivider(cm.Bits & 0xf8 >> 3)
}

func (cm ChipMode608) MarshalJSON() ([]byte, error) {
	return json.Marshal(chipMode608Bits{
		UserExtraAdd:     cm.UserExtraAdd(),
		TTLEnabled:       cm.TTLEnabled(),
		WatchdogDuration: cm.WatchdogDuration(),
		ClockDivider:     cm.ClockDivider(),
	})
}

type SlotConfig struct {
	// Bits1 consists of
	// * ReadKey (4)
	// * NoMac (1)
	// * LimitedUse (1)
	// * EncryptRead (1)
	// * IsSecret (1)
	Bits1 byte
	// Bits2 consists of
	// * WriteKey (4)
	// * WriteConfig (4)
	Bits2 byte
}

type slotConfigBits struct {
	ReadKey     uint16          `json:"read_key"`
	NoMAC       bool            `json:"no_mac"`
	LimitedUse  bool            `json:"limited_use"`
	EncryptRead bool            `json:"encrypt_read"`
	IsSecret    bool            `json:"is_secret"`
	WriteKey    uint16          `json:"write_key"`
	WriteConfig SlotWriteConfig `json:"write_config"`
}

type SlotWriteConfig struct {
	Unknown       bool `json:"unknown"`
	GenKeyEnabled bool `json:"gen_key_enabled"`
	Unknown2      byte `json:"unknown2"`
}

func (sc SlotConfig) ReadKey() uint16 {
	return uint16(sc.Bits1 & 0x0f)
}

func (sc SlotConfig) NoMac() bool {
	return sc.Bits1&0x10 != 0
}

func (sc SlotConfig) LimitedUse() bool {
	return sc.Bits1&0x20 != 0
}

func (sc SlotConfig) EncryptRead() bool {
	return sc.Bits1&0x40 != 0
}

func (sc SlotConfig) IsSecret() bool {
	return sc.Bits1&0x80 != 0
}

func (sc SlotConfig) WriteKey() uint16 {
	return uint16(sc.Bits2 & 0x0f)
}

func (sc SlotConfig) WriteConfig() SlotWriteConfig {
	conf := sc.Bits2 & 0xf0 >> 4
	return SlotWriteConfig{
		Unknown:       conf&0x01 != 0,
		GenKeyEnabled: conf&0x02 != 0,
		Unknown2:      conf >> 2,
	}
}

func (sc SlotConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(slotConfigBits{
		ReadKey:     sc.ReadKey(),
		NoMAC:       sc.NoMac(),
		LimitedUse:  sc.LimitedUse(),
		EncryptRead: sc.EncryptRead(),
		IsSecret:    sc.IsSecret(),
		WriteKey:    sc.WriteKey(),
		WriteConfig: sc.WriteConfig(),
	})
}

// Counter is a monotonic counter.
type Counter struct {
	Value [8]uint8 `json:"value"`
}

type UseLock struct {
	// Bits consists of
	// * UseLockEnable (4)
	// * UseLocKKey (4)
	Bits byte
}

type useLockBits struct {
	UseLockEnable byte `json:"use_lock_enable"`
	UseLockKey    byte `json:"use_lock_key"`
}

func (ul UseLock) UseLockEnable() byte {
	return ul.Bits & 0x0f
}

func (ul UseLock) UseLockKey() byte {
	return ul.Bits & 0xf0 >> 4
}

func (ul UseLock) MarshalJSON() ([]byte, error) {
	return json.Marshal(useLockBits{
		UseLockEnable: ul.UseLockEnable(),
		UseLockKey:    ul.UseLockKey(),
	})
}

type VolatileKeyPermission struct {
	// Bits consists of:
	// * VolatileKeyPermitSlot (4)
	// * Reserved (3)
	// * VolatileKeyPermitEnable (1)
	Bits byte
}

type VolatileKeyPermissionBits struct {
	Slot     byte `json:"slot"`
	Reserved byte `json:"reserved"`
	Enabled  bool `json:"enabled"`
}

func (vkp VolatileKeyPermission) Slot() byte {
	return vkp.Bits & 0x0f
}

func (vkp VolatileKeyPermission) Reserved() byte {
	return (vkp.Bits & 0x70) >> 4
}

func (vkp VolatileKeyPermission) Enabled() bool {
	return vkp.Bits&0x80 != 0
}

func (vkp VolatileKeyPermission) MarshalJSON() ([]byte, error) {
	return json.Marshal(VolatileKeyPermissionBits{
		Slot:     vkp.Slot(),
		Reserved: vkp.Reserved(),
		Enabled:  vkp.Enabled(),
	})
}

type SecureBoot struct {
	// Bits1 consists of
	// * SecureBootMode             2
	// * Reserved0                  1
	// * SecureBootPersistentEnable 1
	// * SecureBootRandNonce        1
	// * Reserved1                  3
	Bits1 byte
	// Bits2 consists of
	// * SecureBootSigDig           4
	// * SecureBootPubKey           4
	Bits2 byte
}

type secureBootBits struct {
	Mode      uint8 `json:"mode"`
	Reserved0 uint8 `json:"reserved0"`

	// PersistentEnabled indicates ifs Secure Boot Persistent Latch is enabled
	//
	// If enabled, the Primary Private Key will be disabled until a valid Secure
	// Boot has occurred.
	PersistentEnabled bool `json:"persistent_enabled"`

	RandNonce bool  `json:"rand_nonce"`
	Reserved1 uint8 `json:"reserved1"`
	SigDig    byte  `json:"sig_dig"`
	PublicKey byte  `json:"public_key"`
}

func (sb SecureBoot) Mode() uint8 {
	return sb.Bits1 & 0x03
}

func (sb SecureBoot) Reserved0() uint8 {
	return (sb.Bits1 & 0x04) >> 2
}

func (sb SecureBoot) PersistentEnabled() bool {
	return sb.Bits1&0x08 != 0
}
func (sb SecureBoot) RandNonce() bool {
	return sb.Bits1&0x10 != 0
}
func (sb SecureBoot) Reserved1() uint8 {
	return sb.Bits1 & 0xe0 >> 4
}
func (sb SecureBoot) SigDig() byte {
	return sb.Bits2 & 0x0f
}

func (sb SecureBoot) PublicKey() byte {
	return sb.Bits2 & 0xf0 >> 4
}

func (sb SecureBoot) MarshalJSON() ([]byte, error) {
	return json.Marshal(secureBootBits{
		Mode:              sb.Mode(),
		Reserved0:         sb.Reserved0(),
		PersistentEnabled: sb.PersistentEnabled(),
		RandNonce:         sb.RandNonce(),
		Reserved1:         sb.Reserved1(),
		SigDig:            sb.SigDig(),
		PublicKey:         sb.PublicKey(),
	})
}

type LockState byte

const (
	// LockStateLocked indicates a locked zone.
	LockStateLocked = LockState(0x00)
	// LockStateUnlocked indicates an unlocked zone.
	LockStateUnlocked = LockState(0x55)
)

func (m LockState) IsLocked() bool {
	return m != LockStateUnlocked
}

func (m LockState) String() string {
	switch m {
	case LockStateLocked:
		return "locked"
	case LockStateUnlocked:
		return "unlocked"
	default:
		return "unknown"
	}
}

func (m LockState) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

type SlotLocked uint16

func (l SlotLocked) IsLocked(slot int) bool {
	if slot >= 16 {
		panic("slot locked contains only 16 slots")
	}
	return int(l)&(1<<slot) == 0
}

func (l SlotLocked) MarshalJSON() ([]byte, error) {
	var slots []bool
	for i := 0; i < 16; i++ {
		slots = append(slots, l.IsLocked(i))
	}
	return json.Marshal(slots)
}

type ChipOptions struct {
	// Bits1 consists of
	// * PowerOnSelfTest       1
	// * IoProtectionKeyEnable 1
	// * KdfAesEnable          1
	// * AutoClearFirstFail    1
	// * Reserved              4
	Bits1 byte
	// Bits2 consists of
	// * EcdhProtectionBits    2
	// * KdfProtectionBits     2
	// * IoProtectionKey       4
	Bits2 byte
}

type chipOptionsBits struct {
	// PowerOnSelfTest enables Power On Self Tests on wake.
	PowerOnSelfTest bool `json:"power_on_self_test"`

	// IoProtectionKeyEnabled indicates if the IO Protection Key is enabled.
	IoProtectionKeyEnabled bool `json:"io_protection_key_enabled"`

	// KdfAesEnable enables the KDF AES feature.
	KdfAesEnable bool `json:"kdf_aes_enabled"`

	// AutoClearFirstFail indicates if the Health Test Failure bit is cleared.
	//
	// If enabled, the Health Test Failure bit is cleared any time that a command
	// fails as a result of a health test failure.
	AutoClearFirstFail bool `json:"auto_clear_first_fail"`

	Reserved byte `json:"reserved"`

	// EcdhProtectionBits indicates if ECDH master secret in the clear is allowed.
	EcdhProtectionBits byte `json:"ecdh_protection_bits"`
	// KdfProtectionBits indicates if KDF functions in the clear is allowed.
	KdfProtectionBits byte `json:"kdf_protection_bits"`

	// IoProtectKey is the slot where the IO Protection Key is found.
	IoProtectionKey byte `json:"io_protection_key"`
}

func (co ChipOptions) PowerOnSelfTest() bool {
	return co.Bits1&0x01 != 0
}

func (co ChipOptions) IoProtectionKeyEnabled() bool {
	return co.Bits1&0x02 != 0
}

func (co ChipOptions) KdfAesEnabled() bool {
	return co.Bits1&0x04 != 0
}

func (co ChipOptions) AutoClearFirstFail() bool {
	return co.Bits1&0x08 != 0
}

func (co ChipOptions) Reserved() byte {
	return co.Bits1 & 0xf0 >> 4
}

func (co ChipOptions) EcdhProtectionBits() byte {
	return co.Bits2 & 0x03
}

func (co ChipOptions) KdfProtectionBits() byte {
	return co.Bits2 & 0x0c >> 2
}

func (co ChipOptions) IoProtectionKey() byte {
	return co.Bits2 & 0xf0 >> 4
}

func (co ChipOptions) MarshalJSON() ([]byte, error) {
	return json.Marshal(chipOptionsBits{
		PowerOnSelfTest:        co.PowerOnSelfTest(),
		IoProtectionKeyEnabled: co.IoProtectionKeyEnabled(),
		KdfAesEnable:           co.KdfAesEnabled(),
		AutoClearFirstFail:     co.AutoClearFirstFail(),
		Reserved:               co.Reserved(),
		EcdhProtectionBits:     co.EcdhProtectionBits(),
		KdfProtectionBits:      co.KdfProtectionBits(),
		IoProtectionKey:        co.IoProtectionKey(),
	})
}

type X509Format struct {
	// Bits consists of
	// * PublicPosition 4
	// * TemplateLength 4
	Bits byte
}

type x509FormatBits struct {
	PublicPosition byte `json:"public_position"`
	TemplateLength byte `json:"template_length"`
}

func (xf X509Format) PublicPosition() byte {
	return xf.Bits & 0x0f
}

func (xf X509Format) TemplateLength() byte {
	return xf.Bits & 0xf0 >> 4
}

func (xf X509Format) MarshalJSON() ([]byte, error) {
	return json.Marshal(x509FormatBits{
		PublicPosition: xf.PublicPosition(),
		TemplateLength: xf.TemplateLength(),
	})
}

type KeyType uint8

const (
	// KeyTypePrivate is a P256 NIST ECC private key.
	KeyTypePrivate = KeyType(0x04)

	// KeyTypeAES is 2 AES 128-bit symmetric keys.
	//
	// Indicates a slot that can store up to 2 AES 128-bit (16 byte) symmetric
	// keys.
	KeyTypeAES = KeyType(0x06)

	// KeyTypeOther can contain any kind of data.
	//
	// This is used by the I/O protection key, Secure Boot and more.
	KeyTypeOther = KeyType(0x07)
)

func (k KeyType) String() string {
	switch k {
	case KeyTypePrivate:
		return "private"
	case KeyTypeAES:
		return "aes"
	case KeyTypeOther:
		return "other"
	default:
		return "unknown"
	}
}

func (k KeyType) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

type KeyConfig struct {
	// Bits1 consists of
	// * Private           1
	// * PubInfo           1
	// * KeyType           3
	// * Lockable          1
	// * ReqRandom         1
	// * ReqAuth           1
	Bits1 byte
	// Bits2 consists of
	// * AuthKey           4
	// * PersistentDisable 1
	// * RFU               1
	// * X509id            2
	Bits2 byte
}

type keyConfigBits struct {
	Private           bool    `json:"private"`
	PubInfo           bool    `json:"pub_info"`
	KeyType           KeyType `json:"key_type"`
	Lockable          bool    `json:"lockable"`
	RequireRandom     bool    `json:"require_random"`
	RequireAuth       bool    `json:"require_auth"`
	AuthKey           byte    `json:"auth_key"`
	PersistentDisable bool    `json:"persistent_disable"`
	RFU               bool    `json:"rfu"`
	X509ID            byte    `json:"x509_id"`
}

func (kc KeyConfig) Private() bool {
	return kc.Bits1&0x01 != 0
}

func (kc KeyConfig) PubInfo() bool {
	return kc.Bits1&0x02 != 0
}

func (kc KeyConfig) KeyType() KeyType {
	return KeyType(kc.Bits1 & 0x1c >> 2)
}

func (kc KeyConfig) Lockable() bool {
	return kc.Bits1&0x20 != 0
}

func (kc KeyConfig) RequireRandom() bool {
	return kc.Bits1&0x40 != 0
}

func (kc KeyConfig) RequireAuth() bool {
	return kc.Bits1&0x80 != 0
}

func (kc KeyConfig) AuthKey() byte {
	return kc.Bits2 & 0x0f
}

func (kc KeyConfig) PersistentDisable() bool {
	return kc.Bits2&0x10 != 0
}

func (kc KeyConfig) RFU() bool {
	return kc.Bits2&0x20 != 0
}

func (kc KeyConfig) X509ID() byte {
	return kc.Bits2 & 0xc0 >> 6
}

func (kc KeyConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(keyConfigBits{
		Private:           kc.Private(),
		PubInfo:           kc.PubInfo(),
		KeyType:           kc.KeyType(),
		Lockable:          kc.Lockable(),
		RequireRandom:     kc.RequireRandom(),
		RequireAuth:       kc.RequireAuth(),
		AuthKey:           kc.AuthKey(),
		PersistentDisable: kc.PersistentDisable(),
		RFU:               kc.RFU(),
		X509ID:            kc.X509ID(),
	})
}

// Config608 represents the configuration used in ATECC608 devices.
type Config608 struct {
	SN03                  [4]byte               `json:"sn03"`
	RevNum                [4]byte               `json:"revision"`
	SN48                  [5]byte               `json:"sn48"`
	AESEnable             AESEnable             `json:"aes_enable"`
	I2CEnable             I2CEnable             `json:"i2c_enable"`
	Reserved15            byte                  `json:"reserved15"`
	I2CAddress            byte                  `json:"i2c_address"`
	Reserved17            byte                  `json:"reserved17"`
	CountMatch            CountMatch            `json:"count_match"`
	ChipMode              ChipMode608           `json:"chip_mode"`
	SlotConfig            [16]SlotConfig        `json:"slot_config"`
	Counter               [2]Counter            `json:"counter"`
	UseLock               UseLock               `json:"use_lock"`
	VolatileKeyPermission VolatileKeyPermission `json:"volatile_key_permission"`
	SecureBoot            SecureBoot            `json:"secure_boot"`
	KdfIvLoc              byte                  `json:"kdf_iv_loc"`
	KdfIvStr              [2]byte               `json:"kdf_iv_str"`
	Reserved68            [9]byte               `json:"reserved68"`
	UserExtra             byte                  `json:"user_extra"`
	UserExtraAdd          byte                  `json:"user_extra_add"`

	// LockValue indicates if the data zone has been locked.
	LockValue LockState `json:"lock_value"`
	// LockConfig indicates if the config zone has been locked.
	LockConfig LockState `json:"lock_config"`

	SlotLocked  SlotLocked    `json:"slot_locked"`
	ChipOptions ChipOptions   `json:"chip_options"`
	X509Format  [4]X509Format `json:"x509_format"`
	KeyConfig   [16]KeyConfig `json:"key_config"`
}

func Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, v)
	return buf.Bytes(), err
}

func Unmarshal(config []byte, data any) error {
	r := bytes.NewReader(config)
	return binary.Read(r, binary.BigEndian, data)
}

func UnmarshalPartial(config []byte, offset int, data any) error {
	var size int
	switch data.(type) {
	case *Config608:
		size = PermanentOffset608 + len(Default608)
	default:
		return errors.New("atecc: unsupported config")
	}

	// TODO: unmarshal w/o allocating all data
	pad := []byte{0x0}
	c := bytes.Repeat(pad, offset)
	c = append(c, config...)
	if len(c) > size {
		return errors.New("atecc: config exceeds maximum size")
	}
	c = append(c, bytes.Repeat(pad, size-len(c))...)
	return Unmarshal(c, data)
}
