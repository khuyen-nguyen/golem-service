package main

func rubyDevDependency() {
	// bundle_version = `grep 'BUNDLED WITH' -A 1 Gemfile.lock`.match(/\d+\.\d+\.\d/).to_s
	// if bundle_version != ""
	//   # Until bundler 2 is default in rubygems, we must install it manually on dev
	//   Gem.install('bundler', "=#{bundle_version}") if `gem list bundler -i -v "=#{bundle_version}"`.strip != 'true'
	// end
	// # Apply to local, when mount empty new /gems
	// system(envs, 'bundle check || bundle install')
}

func nodeDevDependency() {

}
