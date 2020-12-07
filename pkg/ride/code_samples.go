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
a == b`
