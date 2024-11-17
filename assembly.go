package ssce

import (
	"encoding/hex"
	"fmt"
	"strings"
)

var x86asm = `
.code32

entry:
  ret
`

var x64asm = `
.code64

entry:
  // save context and prepare the environment
  {{db .JumpShort}}                          // random jump short
  {{db .SaveContext}}                        // save GP registers
  push rbx                         {{igi}}   // store rbx for save entry address
  push rbp                         {{igi}}   // store rbp for save stack address
  mov rbp, rsp                     {{igi}}   // create new stack frame
  and rsp, 0xFFFFFFFFFFFFFFF0      {{igi}}   // ensure stack is 16 bytes aligned
  sub rsp, 0x200                   {{igi}}   // reserve stack
  fxsave [rsp]                     {{igi}}   // save FP registers

  // calculate the entry address
  {{igi}}                          {{igi}}
  call calc_entry_addr
  flag_CEA:
  {{igi}}                          {{igi}}

  lea rax, [rbx + decoder_stub + 0x02]
  mov word ptr [rax], 0x1234

  // build instructions to stub and erase itself
  {{db .SaveRegister}}
  call decoder_builder             {{igi}}
  call eraser_builder              {{igi}}
  call crypto_key_builder          {{igi}}
  call shellcode_builder           {{igi}}
  call decode_shellcode            {{igi}}
  call erase_builders              {{igi}}
  call erase_decoder_stub          {{igi}}
  call erase_crypto_key_stub       {{igi}}
  {{db .RestoreRegister}}

  // execute the shellcode
  sub rsp, 0x80                    {{igi}}   // reserve stack
  call shellcode_stub              {{igi}}   // call the shellcode
  add rsp, 0x80                    {{igi}}   // restore stack

  // erase the remaining instructions
  push rax                         {{igi}}   // save the shellcode return value
  call erase_shellcode_stub        {{igi}}
  call erase_eraser_stub           {{igi}}
  pop rax                          {{igi}}   // restore the shellcode return value

  fxrstor [rsp]                    {{igi}}   // restore FP registers
  add rsp, 0x200                   {{igi}}   // reserve stack
  mov rsp, rbp                     {{igi}}   // restore stack address
  pop rbp                          {{igi}}   // restore rbp
  pop rbx                          {{igi}}   // restore rbx
  {{db .RestoreContext}}                     // restore GP registers
  ret                              {{igi}}
  
// calculate the shellcode entry address.
calc_entry_addr:
  pop rax                          {{igi}}   // get return address
  mov rbx, rax                     {{igi}}   // calculate entry address
  sub rbx, flag_CEA                {{igi}}   // fix bug for assembler
  push rax                         {{igi}}   // push return address
  ret                              {{igi}}   // return to the entry

decode_shellcode:
  lea rcx, [rbx + shellcode_stub]  {{igi}}
  mov rdx, {{hex .ShellcodeLen}}   {{igi}}
  lea r8, [rbx + crypto_key_stub]  {{igi}}
  mov r9, {{hex .CryptoKeyLen}}    {{igi}}
  sub rsp, 0x40                    {{igi}}
  call decoder_stub                {{igi}}
  add rsp, 0x40                    {{igi}}
  ret                              {{igi}}

erase_builders:
  lea rcx, [rbx + decoder_builder]           {{igi}}
  mov rdx, decoder_stub - decoder_builder    {{igi}}
  call eraser_stub                           {{igi}}
  ret                                        {{igi}}

erase_decoder_stub:
  lea rcx, [rbx + decoder_stub]              {{igi}}
  mov rdx, eraser_stub - decoder_stub        {{igi}}
  call eraser_stub                           {{igi}}
  ret                                        {{igi}}

erase_crypto_key_stub:
  lea rcx, [rbx + crypto_key_stub]           {{igi}}
  mov rdx, shellcode_stub - crypto_key_stub  {{igi}}
  call eraser_stub                           {{igi}}
  ret                                        {{igi}}

erase_shellcode_stub:
  // test rax, rax
  // jmp skip_erase
  lea rcx, [rbx + shellcode_stub]            {{igi}}
  mov rdx, {{hex .ShellcodeLen}}             {{igi}}
  call eraser_stub                           {{igi}}
  skip_erase:
  ret                                        {{igi}}

erase_eraser_stub:
  ret                                        {{igi}}

decoder_builder:
  {{.DecoderBuilder}}
  ret                              {{igi}}

eraser_builder:
  {{.EraserBuilder}}
  ret                              {{igi}}

crypto_key_builder:
  {{.CryptoKeyBuilder}}
  ret                              {{igi}}

shellcode_builder:
  {{.ShellcodeBuilder}}
  ret                              {{igi}}

decoder_stub:
  {{db .DecoderStub}}              {{igi}}

eraser_stub:
  {{db .EraserStub}}               {{igi}}

crypto_key_stub:
  {{db .CryptoKeyStub}}            {{igi}}

shellcode_stub:
  {{db .ShellcodeStub}}
`

type asmContext struct {
	JumpShort      []byte
	SaveContext    []byte
	RestoreContext []byte

	SaveRegister     []byte
	RestoreRegister  []byte
	DecoderBuilder   string
	EraserBuilder    string
	CryptoKeyBuilder string
	ShellcodeBuilder string

	DecoderStub   []byte
	EraserStub    []byte
	CryptoKeyStub []byte
	CryptoKeyLen  int
	ShellcodeStub []byte
	ShellcodeLen  int
}

func toDB(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	builder := strings.Builder{}
	builder.WriteString(".byte ")
	for i := 0; i < len(b); i++ {
		builder.WriteString("0x")
		builder.WriteString(hex.EncodeToString([]byte{b[i]}))
		builder.WriteString(", ")
	}
	return builder.String()
}

func toHex(v int) string {
	return fmt.Sprintf("0x%X", v)
}
