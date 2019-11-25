import gen from 'faker'

let gooduser = () => ({
  username: "__test__" + gen.internet.userName(),
  password: gen.internet.password(),
  phone: "(208)" + gen.phone.phoneNumberFormat(1).slice(5),
})

let goodnote = () => ({
  text: gen.lorem.paragraph(),
})

context('smscp', () => {
  beforeEach(() => {
    cy.visit('http://localhost:3000')
  })

  describe('user', () => {
    let user = gooduser()

    it('register', () => {
      cy.get('#register input[name=Username]').type(user.username)
      cy.get('#register input[name=Password]').type(user.password)
      cy.get('#register input[name=Verify]').type(user.password)
      cy.get('#register input[name=Phone]').type(user.phone)
      cy.get('#register').submit()
      cy.get('h3').should('contain', 'Welcome back')
    })

    it('login and logout', () => {
      // login
      cy.get('#login input[name=Username]').type(user.username)
      cy.get('#login input[name=Password]').type(user.password)
      cy.get('#login').submit()
      cy.get('h3').should('contain', 'Welcome back')
      // logout
      cy.get("form[action='/user/logout']").submit()
      cy.get('h3').should('contain', 'Get started')
    })

    it('invalid verify', () => {
      cy.get('#register input[name=Username]').type(user.username)
      cy.get('#register input[name=Password]').type(user.password)
      cy.get('#register input[name=Verify]').type(gen.internet.password())
      cy.get('#register input[name=Phone]').type(user.phone)
      cy.get('#register').submit()
      cy.get('h1').should('contain', 'Error')
    })

    it('invalid phone', () => {
      cy.get('#register input[name=Username]').type(user.username)
      cy.get('#register input[name=Password]').type(user.password)
      cy.get('#register input[name=Verify]').type(user.password)
      cy.get('#register input[name=Phone]').type(gen.phone.phoneNumber())
      cy.get('#register').submit()
      cy.get('h1').should('contain', 'Error')
    })

    it('username already taken', () => {
      let diff = gooduser()
      cy.get('#register input[name=Username]').type(user.username)
      cy.get('#register input[name=Password]').type(diff.password)
      cy.get('#register input[name=Verify]').type(diff.password)
      cy.get('#register input[name=Phone]').type(diff.phone)
      cy.get('#register').submit()
      cy.get('h1').should('contain', 'Error')
    })

    it('phone already taken', () => {
      let diff = gooduser()
      cy.get('#register input[name=Username]').type(diff.username)
      cy.get('#register input[name=Password]').type(diff.password)
      cy.get('#register input[name=Verify]').type(diff.password)
      cy.get('#register input[name=Phone]').type(user.phone)
      cy.get('#register').submit()
      cy.get('h1').should('contain', 'Error')
    })

    it('update', () => {
      let next = gooduser()
      // login
      cy.get('#login input[name=Username]').type(user.username)
      cy.get('#login input[name=Password]').type(user.password)
      cy.get('#login').submit()
      cy.get('h3').should('contain', 'Welcome back')
      // update
      cy.get('#update input[name=Username]').type(next.username)
      cy.get('#update input[name=Password]').type(next.password)
      cy.get('#update input[name=Verify]').type(next.password)
      cy.get('#update input[name=Phone]').type(next.phone)
      cy.get('#update').submit()
      cy.get('h3').should('contain', 'Welcome back')
      // logout
      cy.get("form[action='/user/logout']").submit()
      cy.get('h3').should('contain', 'Get started')
      // login
      user = next
      cy.get('#login input[name=Username]').type(user.username)
      cy.get('#login input[name=Password]').type(user.password)
      cy.get('#login').submit()
      cy.get('h3').should('contain', 'Welcome back')
    })

    it('delete', () => {
      // login
      cy.get('#login input[name=Username]').type(user.username)
      cy.get('#login input[name=Password]').type(user.password)
      cy.get('#login').submit()
      cy.get('h3').should('contain', 'Welcome back')
      // delete
      cy.get("#delete").submit()
      cy.get('h3').should('contain', 'Get started')
      // login x2
      cy.get('#login input[name=Username]').type(user.username)
      cy.get('#login input[name=Password]').type(user.password)
      cy.get('#login').submit()
      cy.get('h1').should('contain', 'Error')
    })

  })

  describe('note', () => {
    let user = gooduser()
    let note = goodnote()

    it('create', () => {
      // register user
      cy.get('#register input[name=Username]').type(user.username)
      cy.get('#register input[name=Password]').type(user.password)
      cy.get('#register input[name=Verify]').type(user.password)
      cy.get('#register input[name=Phone]').type(user.phone)
      cy.get('#register').submit()
      cy.get('h3').should('contain', 'Welcome back')
      // create note
      cy.get('#create input[name=Text]').type(note.text)
      cy.get('#create').submit()
      cy.get('h3').should('contain', 'Welcome back')
    })

    // Invalid test as form can't submit on no input. How do I specifically test
    // for this with Cypress?
    // it('invalid', () => {
    //   // login user
    //   cy.get('#login input[name=Username]').type(user.username)
    //   cy.get('#login input[name=Password]').type(user.password)
    //   cy.get('#login').submit()
    //   cy.get('h3').should('contain', 'Welcome back')
    //   // create bad note
    //   // cy.get('#create input[name=Text]').type('') // Cypress doesn't do zero
    //   // length strings.
    //   cy.get('#create').submit()
    //   cy.get('h1').should('contain', 'Error')
    // })

  })

})
