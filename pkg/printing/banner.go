package printing

import "github.com/emrekara369/dalfox_new/v2/pkg/model"

// Banner is DalFox banner function
func Banner(options model.Options) {
	DalLog("", `
    _..._
  .' .::::.   __   _   _    ___ _ __ __
 :  :::::::: |  \ / \ | |  | __/ \\ V /
 :  :::::::: | o ) o || |_ | _( o )) (
 '. '::::::' |__/|_n_||___||_| \_//_n_\
   '-.::''    
`, options)
	DalLog("", "🌙🦊 Dalfox is a powerful open-source XSS scanner and utility focused on automation.", options)
}
