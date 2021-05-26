package ride

const fcall1 = `
func getInt(key: String) = {
    match getInteger(this, key) {
        case x : Int => x
        case _ => 0
    }
}

let a = getInt("5")
let b = getInt("6")
a == b
`

const finf = `
func abc() = {
    func in() = {
        true
    }
    in()
}
abc()
`

const intersectNames = `
{-# STDLIB_VERSION 3 #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# CONTENT_TYPE EXPRESSION #-}
func inc(v: Int) = v + 1
func call(inc: Int) = {
    inc(inc)
}
call(2) == 3
`
