FROM ruby:3

RUN gem install httparty

ADD test-standard.rb /test-standard.rb
RUN chmod +x /test-standard.rb

ADD test-reverse.rb /test-reverse.rb
RUN chmod +x /test-reverse.rb